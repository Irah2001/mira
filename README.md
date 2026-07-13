# Mira — Outil de Mémoire Personnelle (Version 2.0)

Mira est une base de connaissances locale et centralisée qui vous permet de stocker, lister, modifier et rechercher vos notes ou mémos personnels. 

Cette version 2 (V2) apporte une architecture client-serveur complète. Elle intègre désormais une **API HTTP REST v1** hautement robuste tout en conservant l'accès historique via l'**interface CLI**, les deux interfaces partageant de manière asynchrone le même fichier de stockage persistant grâce à un mécanisme de verrous concurrents (`sync.RWMutex`).

---

## Architecture & Organisation du Dépôt

Le projet est structuré selon les standards modernes de l'écosystème Go :

```text
mira/
├── cmd/
│   ├── cli/
│   │   └── main.go                 # Point d'entrée de l'application CLI locale
│   └── api/
│       └── main.go                 # Point d'entrée du serveur API HTTP REST
├── internal/
│   ├── notes/
│   │   ├── note.go                 # Modèles de domaine, payloads et validations
│   │   ├── store.go                # Contrat/Interface du NoteStore
│   │   └── jsonl.go                # Implémentation du stockage JSON Lines (Thread-safe)
│   ├── search/
│   │   └── search.go               # Algorithme de recherche textuelle naïve
│   └── http/
│       └── handlers/
│           ├── middleware.go       # Middlewares (Request ID, Slog, Recovery)
│           ├── notes.go            # Contrôleurs / Gestionnaires d'endpoints
│           ├── notes_test.go       # Tests unitaires des handlers HTTP
│           └── response.go         # Structure de l'enveloppe JSON standardisée
├── go.mod                          # Fichier de définition du module Go
└── README.md                       # Cette documentation
```

## Spécification du stockage JSON Lines

- Format : JSON Lines (`.jsonl`), un objet JSON valide par ligne.

- Emplacement par défaut : `~/.mira/notes.jsonl` (créé automatiquement s'il n'existe pas).

- Concurrence : L'accès disque est protégé par un verrou de lecture/écriture global (sync.RWMutex) permettant des lectures parallèles instantanées depuis l'API tout en bloquant l'accès lors des écritures, mises à jour ou suppressions pour éviter toute corruption de données.

## Utilisation de l'Interface CLI (Locale)

L'accès en ligne de commande permet des interactions rapides en local sur votre machine.

1. Ajouter une note :
   ```bash
   ./mira add "Titre de la note" "Contenu de la note"
   ```

2. Lister toutes les notes :
   ```bash
    ./mira list
    ```

3. Rechercher des notes par mot-clé :
    ```bash
    ./mira search "mot-clé"
    ```

## Utilisation de l'API HTTP REST

Le serveur API s'exécute par défaut sur le port :8080. Pour le lancer, utilisez la commande suivante :

```bash
go run cmd/api/main.go
```

### Enveloppe de Réponse JSON Stable
Afin de garantir la stabilité des clients (Web, Mobile, Extension), toutes les réponses HTTP sans exception adoptent une structure JSON unifiée :

```json
{
  "success": true,
  "data": { ... },
  "error": null
}
```

### Référence des Endpoints & Exemples de Requêtes:

| Méthode | Endpoint | Description | Exemple curl |
|---------|----------|-------------|---------------|
| GET     | api/v1/notes   | Récupère toutes les notes | `curl http://localhost:8080/api/v1/notes` |
| GET     | api/v1/notes/{id} | Récupère une note spécifique par ID | `curl http://localhost:8080/api/v1/notes/1` |
| POST    | api/v1/notes   | Crée une nouvelle note | `curl -X POST http://localhost:8080/api/v1/notes -H "Content-Type: application/json" - d '{"title":"Nouvelle Note","content":"Contenu de la note"}'` |
| PUT     | api/v1/notes/{id} | Met à jour une note spécifique par ID | `curl -X PUT http://localhost:8080/api/v1/notes/1 -H "Content-Type: application/json" -d '{"title":"Titre mis à jour","content":"Contenu mis à jour"}'` |
| DELETE  | api/v1/notes/{id} | Supprime une note spécifique par ID | `curl -X DELETE http://localhost:8080/api/v1/notes/1` |
| GET     | api/v1/search | Recherche des notes par mot-clé | `curl http://localhost:8080/api/v1/search\?q=modules` |

### Documentation OpenAPI/Swagger

La documentation OpenAPI est disponible à l'adresse suivante : [http://localhost:8080/docs/](http://localhost:8080/docs/)
