package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"mira/internal/notes"

	"github.com/joho/godotenv"
)

type APIResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   any             `json:"error"`
}

func main() {
	godotenv.Load()

	apiURL := os.Getenv("MIRA_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080/api/v1"
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	client := &http.Client{Timeout: 5 * time.Second}

	switch command {
	case "add":
		if len(os.Args) < 4 {
			fmt.Println("Erreur : mira add requiert un titre et un contenu.")
			printUsage()
			os.Exit(1)
		}

		payload := map[string]any{
			"title":   os.Args[2],
			"content": os.Args[3],
			"tags":    []string{"cli"},
		}
		body, _ := json.Marshal(payload)

		resp, err := client.Post(apiURL+"/notes", "application/json", bytes.NewBuffer(body))
		if err != nil || resp.StatusCode != http.StatusCreated {
			fmt.Printf("❌ Erreur lors de l'appel à l'API: %v\n(Vérifie que le serveur est bien lancé sur %s !)\n", err, apiURL)
			os.Exit(1)
		}
		fmt.Println("✅ Note envoyée à l'API avec succès. (L'enrichissement démarre en arrière-plan !)")

	case "list":
		resp, err := client.Get(apiURL + "/notes?limit=10")
		if err != nil {
			fmt.Printf("❌ Impossible de contacter l'API sur %s\n", apiURL)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var result APIResponse
		json.NewDecoder(resp.Body).Decode(&result)

		var noteList []notes.Note
		json.Unmarshal(result.Data, &noteList)

		if len(noteList) == 0 {
			fmt.Println("Aucune note pour le moment.")
			return
		}

		fmt.Println("📝 Vos 10 dernières notes :")
		for _, n := range noteList {
			status := "⏳"
			switch n.EnrichmentStatus {
			case "done":
				status = "✅"
			case "failed":
				status = "❌"
			}
			fmt.Printf("\n[%s] %s %s (Score: %d)\n> %s\n", n.CreatedAt.Format("2006-01-02 15:04"), status, n.Title, n.Score, n.Content)
		}

	case "search":
		if len(os.Args) < 3 {
			fmt.Println("Erreur : veuillez fournir un terme de recherche.")
			os.Exit(1)
		}
		query := os.Args[2]

		resp, err := client.Get(apiURL + "/search?q=" + query)
		if err != nil {
			fmt.Printf("❌ Impossible de contacter l'API sur %s\n", apiURL)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var result APIResponse
		json.NewDecoder(resp.Body).Decode(&result)

		var searchResults []notes.Note
		json.Unmarshal(result.Data, &searchResults)

		if len(searchResults) == 0 {
			fmt.Printf("Aucun résultat pour la recherche : %q\n", query)
			return
		}

		fmt.Printf("🔍 Résultats de recherche pour %q (%d trouvé(s)) :\n", query, len(searchResults))
		for _, n := range searchResults {
			fmt.Printf("\n- %s (Score pertinence: %d)\n  %s\n", n.Title, n.Score, n.Content)
		}

	default:
		fmt.Printf("Erreur : commande inconnue '%s'\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("\nUtilisation de mira (Mode API Client) :")
	fmt.Println("  mira add \"titre\" \"contenu\"   : Ajoute une nouvelle note via l'API")
	fmt.Println("  mira list                    : Affiche les 10 dernières notes via l'API")
	fmt.Println("  mira search <texte>          : Recherche avancée via l'API")
}
