package main

import (
	"context"
	"fmt"
	"os"

	"mira/internal/notes"
	"mira/internal/search"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("Erreur: La variable DATABASE_URL n'est pas définie dans l'environnement ou le fichier .env")
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		fmt.Printf("Erreur : impossible de se connecter à la BDD: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	store := notes.NewPostgresStore(pool)

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

		note := notes.NewNote(title, content, []string{})

		if err := store.Add(note); err != nil {
			fmt.Printf("Erreur lors de la sauvegarde : %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Note ajoutée avec succès dans PostgreSQL.")

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
