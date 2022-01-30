package main

import (
	"errors"
	"fmt"
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
	ip       string
	port     uint
	savePath string
)

type Save struct {
	Filename    string
	DownloadUrl string
	ViewUrl     string
	Type        string
	Timestamp   time.Time
	SaveTime    string
}

type Game struct {
	Name  string
	Saves []Save
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

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	http.HandleFunc("/", index)

	http.ListenAndServe(fmt.Sprintf("%s:%d", ip, port), nil)

	return nil
}

func index(w http.ResponseWriter, req *http.Request) {
	template := template.Must(template.ParseFiles("templates/list.html"))
	listData := ListData{Games: getGameData(w)}
	template.Execute(w, listData)
}

func getGameData(w http.ResponseWriter) []Game {
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

		game, found := gameMap[gameName]
		if !found {
			game = Game{Name: gameName, Saves: []Save{}}
		}

		downloadUrl := "foobar"

		game.Saves = append(game.Saves, Save{
			Filename:    fileName,
			DownloadUrl: downloadUrl,
			ViewUrl:     fmt.Sprintf("https://satisfactory-calculator.com/?url=%s", downloadUrl),
			Type:        saveType,
			Timestamp:   timestamp,
			SaveTime:    saveTime,
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
