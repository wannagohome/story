package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/anthropics/story/internal/client"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "story",
		Short: "Story - AI-powered narrative mystery game",
		Long:  "A multiplayer terminal-based narrative mystery game powered by AI.",
	}

	hostCmd := &cobra.Command{
		Use:   "host",
		Short: "Start a new game session and join as host",
		RunE: func(cmd *cobra.Command, args []string) error {
			port, _ := cmd.Flags().GetInt("port")
			nickname, _ := cmd.Flags().GetString("nickname")

			// In a full implementation, this would start the server process.
			// For now, we just start the TUI client pointing at localhost.
			serverURL := fmt.Sprintf("ws://localhost:%d/ws", port)

			cfg := client.ClientConfig{
				ServerURL: serverURL,
				RoomCode:  "", // Server assigns room code
				Nickname:  nickname,
				IsHost:    true,
			}

			return runClient(cfg)
		},
	}
	hostCmd.Flags().Int("port", 8080, "WebSocket server port")
	hostCmd.Flags().String("nickname", "", "Your display name")

	joinCmd := &cobra.Command{
		Use:   "join <room-code>",
		Short: "Join an existing game session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			roomCode := args[0]
			server, _ := cmd.Flags().GetString("server")
			nickname, _ := cmd.Flags().GetString("nickname")

			if server == "" {
				server = "ws://localhost:8080/ws"
			}

			cfg := client.ClientConfig{
				ServerURL: server,
				RoomCode:  roomCode,
				Nickname:  nickname,
				IsHost:    false,
			}

			return runClient(cfg)
		},
	}
	joinCmd.Flags().String("server", "", "WebSocket server URL (e.g., ws://192.168.1.5:8080/ws)")
	joinCmd.Flags().String("nickname", "", "Your display name")

	rootCmd.AddCommand(hostCmd, joinCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runClient(cfg client.ClientConfig) error {
	model := client.NewAppModel(cfg)
	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}
