
[English](CONTRIBUTING.md) | **Français**

## Contribution

Si vous connaissez un site qui pourrait être utile à d'autres utilisateurs, l'implémentation d'un nouveau provider est l'une des meilleures façons de contribuer au projet.

### Ajouter un nouveau provider

Chaque implémentation spécifique à un site se trouve dans :

```text
internal/source/
```

Par exemple :

```text
internal/source/
├── animesama/
├── mangafreak/
└── yourprovider/
```

Un provider doit implémenter l'interface commune suivante :

```go
type Provider interface {
	Name() string

	Search(
		query string,
		log *logger.Logger,
	) ([]SearchResult, error)

	ListScanPaths(
		workURL string,
		log *logger.Logger,
	) ([]common.SelectableItem, error)

	ListEntries(
		workURL string,
		scanPath string,
		log *logger.Logger,
	) (Work, []Entry, error)

	GetPageImageURLs(
		entryURL string,
		log *logger.Logger,
	) ([]string, error)
}
```

Le workflow d'implémentation habituel est le suivant :

```text
Recherche
  ↓
Sélection du manga
  ↓
ListScanPaths
  ↓
Sélection du scan / de la version
  ↓
ListEntries
  ↓
Sélection des chapitres
  ↓
GetPageImageURLs
  ↓
Téléchargement des images
  ↓
Sortie PDF / ebook-friendly
```

### 1. Implémenter la recherche de manga

`Search()` doit effectuer une recherche dans le catalogue du provider et retourner des résultats contenant au minimum :

```go
SearchResult{
	Title: "...",
	URL:   "...",
}
```

L'URL retournée doit identifier le manga et pouvoir être utilisée par les méthodes suivantes du provider.

Si le site expose une page de recherche HTML classique, les helpers déjà disponibles dans `internal/source/common` peuvent éventuellement prendre en charge une partie de la logique de parsing.

### 2. Implémenter la détection des scans / versions

`ListScanPaths()` doit retourner les différentes versions disponibles pour le manga sélectionné.

Par exemple :

```text
Scans
Colored
Black & White
English
French
```

Si le provider ne propose qu'une seule version, il suffit de retourner un seul élément sélectionnable automatiquement.

Par exemple :

```go
[]common.SelectableItem{
	{
		Label: "Your Provider",
		Value: workURL,
	},
}
```

L'application ignore automatiquement l'écran de sélection lorsqu'une seule option est disponible.

### 3. Implémenter la détection des chapitres

`ListEntries()` doit retourner les informations concernant l'œuvre sélectionnée ainsi que ses entrées téléchargeables.

Pour des chapitres de manga, le `Work` retourné doit généralement utiliser :

```go
Work{
	Title: "Manga Name",
	Kind:  ItemChapter,
}
```

Chaque chapitre doit ensuite être converti en `Entry` :

```go
Entry{
	Number: 100,
	Label:  "Chapter 100",
	URL:    "https://example.com/chapter/100",
}
```

Le code générique de l'application gère le filtrage des plages de chapitres. L'implémentation du provider doit donc normalement retourner la liste complète des chapitres disponibles.

### 4. Implémenter la détection des images d'un chapitre

`GetPageImageURLs()` reçoit l'URL d'une entrée et doit retourner la liste ordonnée des images appartenant à ce chapitre.

Par exemple :

```go
[]string{
	"https://example.com/chapter/100/001.jpg",
	"https://example.com/chapter/100/002.jpg",
	"https://example.com/chapter/100/003.jpg",
}
```

La manière de récupérer ces URLs dépend entièrement du site.

Certains providers peuvent utiliser des URLs d'images séquentielles, tandis que d'autres nécessitent de parser la page HTML du chapitre.

Une fois la liste retournée, le downloader générique se charge du téléchargement réel des images.

### 5. Enregistrer le provider

Une fois le provider implémenté, ajoutez-le à la registry centrale des providers dans `internal/source`.

Cette registry est également utilisée par le CLI et le TUI afin de déterminer quels providers sont disponibles. Les providers ne doivent donc pas être codés en dur séparément dans les interfaces utilisateur.

Un enregistrement ressemble généralement à ceci :

```go
"yourprovider": {
	Label: "Your Provider",
	New: func(customDomain string) sourcetypes.Provider {
		return yourprovider.New(customDomain)
	},
},
```

Une fois enregistré, le provider devrait automatiquement devenir disponible via :

```text
goweeb --source yourprovider ...
```

ainsi que dans l'écran de sélection des providers du TUI.

### 6. Tester le workflow complet

Avant d'ouvrir une pull request, vérifiez au minimum le workflow suivant :

```text
Recherche
→ sélection du manga
→ sélection du scan / de la version
→ détection des chapitres
→ filtrage de la plage
→ détection des pages
→ téléchargement des images
→ sortie finale
```

Testez également les cas suivants :

* la recherche ne retourne aucun résultat ;
* la recherche retourne un seul résultat ;
* la recherche retourne plusieurs résultats ;
* une seule version de scan est disponible ;
* le site retourne une erreur HTTP ;
* un chapitre ne contient aucune image.

Enfin, exécutez :

```bash
gofmt -w .
go test ./...
```

et vérifiez que le projet compile correctement.

### Recommandations pour l'implémentation d'un provider

Lors de l'ajout d'un scraper :

* Gardez le parsing spécifique au site dans le package du provider concerné.
* Réutilisez autant que possible les helpers disponibles dans `internal/source/common` et `internal/fetch`.
* Évitez de dupliquer la logique générique de téléchargement ou de sélection.
* Ne codez pas les providers en dur dans le CLI ou le TUI.
* Préférez les URLs extraites directement depuis le site plutôt que de les reconstruire lorsque cela est possible.
* Retournez des erreurs claires lorsque la structure du site ne correspond plus à ce que le scraper attend.
* Conservez le support d'un domaine personnalisé lorsque le provider peut raisonnablement fonctionner avec des miroirs ou des changements de domaine.

La structure des sites web change fréquemment. Garder chaque scraper isolé dans son propre provider facilite donc grandement la maintenance.
