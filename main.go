package main

import (
	"fmt"
	"os"
	"path/filepath"

	"mira/internal/notes"
	"mira/internal/search"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Erreur : impossible de trouver le dossier utilisateur.")
		os.Exit(1)
	}

	miraDir := filepath.Join(homeDir, ".mira")
	if err := os.MkdirAll(miraDir, 0755); err != nil {
		fmt.Printf("Erreur lors de la création du dossier %s: %v\n", miraDir, err)
		os.Exit(1)
	}

	storePath := filepath.Join(miraDir, "notes.jsonl")
	store := notes.NewJSONLStore(storePath)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "add":
		if len(os.Args) < 4 {
			fmt.Println("Erreur : mira add requiert un titre et un contenu.")
			printUsage()
			os.Exit(1)
		}
		title := os.Args[2]
		content := os.Args[3]

		note := notes.NewNote(title, content)
		if err := store.Add(note); err != nil {
			fmt.Printf("Erreur lors de la sauvegarde : %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Note ajoutée avec succès.")

	case "list":
		recentNotes, err := store.List(10)
		if err != nil {
			fmt.Printf("Erreur lors de la lecture : %v\n", err)
			os.Exit(1)
		}

		if len(recentNotes) == 0 {
			fmt.Println("Aucune note pour le moment.")
			return
		}

		fmt.Println("📝 Vos 10 dernières notes :")
		for _, n := range recentNotes {
			fmt.Printf("\n[%s] %s\n> %s\n", n.CreatedAt.Format("2006-01-02 15:04"), n.Title, n.Content)
		}

	case "search":
		if len(os.Args) < 3 {
			fmt.Println("Erreur : veuillez fournir un terme de recherche.")
			os.Exit(1)
		}
		query := os.Args[2]

		allNotes, err := store.GetAll()
		if err != nil {
			fmt.Printf("Erreur lors de la lecture : %v\n", err)
			os.Exit(1)
		}

		results := search.Search(allNotes, query)
		if len(results) == 0 {
			fmt.Printf("Aucun résultat pour la recherche : %q\n", query)
			return
		}

		fmt.Printf("🔍 Résultats de recherche pour %q (%d trouvé(s)) :\n", query, len(results))
		for _, n := range results {
			fmt.Printf("\n- %s\n  %s\n", n.Title, n.Content)
		}

	default:
		fmt.Printf("Erreur : commande inconnue '%s'\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("\nUtilisation de mira :")
	fmt.Println("  mira add \"titre\" \"contenu\"   : Ajoute une nouvelle note")
	fmt.Println("  mira list                    : Affiche les 10 dernières notes")
	fmt.Println("  mira search <texte>          : Recherche dans le titre et le contenu")
}
