package main

import (
	"embed"
	"errors"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"html/template"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var (
	ip           string
	port         uint
	savePath     string
	printVersion bool

	version string

	//go:embed templates
	templates embed.FS

	pages = map[string]string{
		"list": "templates/list.html",
	}
)

type Save struct {
	Filename    string
	DownloadUrl string
	ViewUrl     string
	FullUrl			string
	Type        string
	Timestamp   time.Time
	SaveTime    string
	Filesize    string
}

type Game struct {
	Name  						string
	Saves 						[]Save
	LatestDownloadUrl	string
	LatestViewUrl			string
}

type ListData struct {
	Games []Game
}

func main() {
	cmd := configureCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Println("Failed run")
		fmt.Println(err)
		os.Exit(1)
	}
}

func configureCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "satisfactory [flags] save-directory",
		Short: "HTTP server to list Satisfactory saves and link to download or view them in satisfactory calculator.",
		Args: func(cmd *cobra.Command, args []string) error {
			if printVersion {
				return nil
			}

			if len(args) < 1 {
				return errors.New("The path to your Satisfactory saves directory is required")
			}

			savePath = args[0]
			_, err := os.Stat(savePath)
			if os.IsNotExist(err) {
				return errors.New("Save path does not exist.")
			}

			return nil
		},
		RunE: run,
	}

	cmd.Flags().UintVarP(&port,
		"port",
		"p",
		1234,
		"The port to listen on for HTTP requests")

	cmd.Flags().StringVarP(&ip,
		"ip",
		"i",
		"",
		"The ip address to listen on for HTTP requests")

	cmd.Flags().BoolVarP(&printVersion,
		"version",
		"v",
		false,
		"Print the current version")

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	if printVersion {
		fmt.Println(version)
		return nil
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/latest/", latest)

	saveServer := http.FileServer(http.Dir(savePath))
	http.Handle("/saves/", http.StripPrefix("/saves", saveServer))

	http.ListenAndServe(fmt.Sprintf("%s:%d", ip, port), nil)

	return nil
}

func index(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		w.WriteHeader(404)
		fmt.Fprintln(w, "404 Page Not Found")
		return
	}

	listPage, err := template.ParseFS(templates, pages["list"])
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, "500 Internal Server Error")
		return
	}

	httpHost := req.Host
	listData := ListData{Games: getGameData(httpHost)}

	listPage.Execute(w, listData)
}

func latest(w http.ResponseWriter, req *http.Request) {
	name := strings.TrimPrefix(req.URL.Path, "/latest/")
	if (name == "") {
		w.WriteHeader(404)
		fmt.Fprintln(w, "404 Page Not Found")
		return
	}

	games := getGameData(req.Host)
	for _, game := range games {
		if game.Name != name {
			continue
		}

		w.Header().Set("Location", game.Saves[0].FullUrl)
		w.WriteHeader(302)
		return
	}

	w.WriteHeader(404)
	fmt.Fprintln(w, "404 Page Not Found")
	return
}

func getGameData(httpHost string) []Game {
	var gameMap = make(map[string]Game)

	saves, _ := filepath.Glob(filepath.Join(savePath, "*.sav"))
	for _, save := range saves {
		fileName := path.Base(save)
		saveName := strings.ReplaceAll(fileName, ".sav", "")
		parts := strings.Split(saveName, "_")
		if len(parts) < 2 {
			continue
		}

		stats, err := os.Stat(save)
		if err != nil {
			continue
		}

		gameName := parts[0]
		saveType := parts[1]
		timestamp := stats.ModTime()
		saveTime := timestamp.Format("Mon, 02 Jan 2006 15:04:05")
		filesize := humanize.Bytes(uint64(stats.Size()))

		game, found := gameMap[gameName]
		if !found {
			latestUrl := fmt.Sprintf("%s://%s/latest/%s", "https", httpHost, gameName)
			latestViewUrl := fmt.Sprintf("https://satisfactory-calculator.com/en/interactive-map?url=%s", latestUrl)

			game = Game{
				Name:								gameName,
				LatestDownloadUrl:	latestUrl,
				LatestViewUrl:     	latestViewUrl,
				Saves: 							[]Save{},
			}
		}

		downloadUri := fmt.Sprintf("/saves/%s", fileName)
		fullUrl := fmt.Sprintf("%s://%s%s", "https", httpHost, downloadUri)

		game.Saves = append(game.Saves, Save{
			Filename:    fileName,
			DownloadUrl: downloadUri,
			ViewUrl:     fmt.Sprintf("https://satisfactory-calculator.com/en/interactive-map?url=%s", fullUrl),
			FullUrl:		 fullUrl,
			Type:        saveType,
			Timestamp:   timestamp,
			SaveTime:    saveTime,
			Filesize:    filesize,
		})

		gameMap[gameName] = game
	}

	gameSlice := []Game{}
	for _, game := range gameMap {
		sort.Slice(game.Saves, func(a, b int) bool {
			return game.Saves[a].Timestamp.After(game.Saves[b].Timestamp)
		})
		gameSlice = append(gameSlice, game)
	}

	return gameSlice
}
