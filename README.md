**French** | [English](README_EN.md)

# goweeb-sama

![goweeb TUI demo](./assets/demo.gif)

**goweeb-sama** est un outil très rapide permettant de télécharger proprement les scans de ton manga préféré
depuis le site **anime-sama** (qui fait un excellent travail d'ailleurs, merci à vous).

## Installation

L'installation est très simple, il suffit d'aller dans l'onglet **[Releases](https://github.com/EliasLd/goweeb-sama/releases/latest)** et de télécharger la dernière version **TUI** de l'outil pour ton système d'exploitation.

### Windows
1. Télécharge l'archive pour windows
2. Place l'exécutable dans un dossier de ton choix
3. Ouvre un terminal (PowerShell ou CMD) dans ce dossier et utilise l'outil

### Linux / macOS
1. Télécharge l'archive correspondant à ton système (*darwin* pour macos)
2. Rends-le exécutable : `chmod +x goweeb`
3. (Optionnel) Déplace-le dans `/usr/local/bin` pour l'utiliser depuis n'importe où

## Utilisation

**goweeb-sama** est disponible en deux versions :

### Interface graphique (TUI) - Recommandé pour les débutants

Lance simplement l'exécutable TUI :

```bash
# Linux / macOS
./goweeb

# Windows
goweeb.exe
```

Laisse-toi guider par l'interface interactive !

- Saisis le nom du manga que tu recherches
- Choisis parmi les résultats trouvés
- Sélectionne la version (couleur/noir et blanc, VF/VA)
- Configure tes options de téléchargement
- Lance le téléchargement et suis la progression en temps réel

### Ligne de commande (CLI) - Pour les utilisateurs avancés

### Format du nom de manga

**Important** : Le nom du manga doit être entré **entre guillemets**. Tu peux utiliser n'importe quel terme de recherche (titre français, anglais, ou japonais).

Exemples :
- `"one piece"`
- `"jujutsu kaisen"`
- `"chainsaw man"`
- `"attaque des titans"`

Si plusieurs résultats correspondent à ta recherche, l'outil te proposera de choisir le bon manga dans une liste interactive.

### Syntaxe de base

```bash
goweeb [options] "nom-du-manga"
```

### Options disponibles

| Option | Raccourci | Description |
|--------|-----------|-------------|
| `--all` | `-a` | Télécharge tous les chapitres disponibles |
| `--scan-dir <dossier>` | `-d` | Dossier où sauvegarder les PDF (par défaut : `pdf`) |
| `--keep-images` | `-k` | Garde les images après la création du PDF |
| `--domain <url>` | `-u` | Remplace le domaine anime-sama (ex: `https://anime-sama.tv`) |
| `--ebook-friendly ` | aucun | Télécharge les chapitres au format image dans des dossiers  séparés (e.g. Chapter 001/, Chapter 002/). Compatible avec [Kindle Comic Converter](https://github.com/ciromattia/kcc). |
| `--debug ` | aucun | Active le mode debug pour avoir plus de logs (CLI uniquement) |

### Exemples d'utilisation

#### Télécharger tous les chapitres d'un manga
```bash
goweeb --all "one piece"
# ou
goweeb -a "jujutsu kaisen"
```

#### Télécharger une plage de chapitres

Lorsque le manga a été choisi, il **te sera demandé** d'entrer une plage de chapitres à télécharger.
Suit simplement les instructions et choisit la plage de chapitres que tu souhaites.

#### Spécifier un dossier de destination
```bash
# Les PDF seront sauvegardés dans le dossier "mes-mangas"
goweeb -d mes-mangas --all "naruto"

# Ou avec le chemin complet (sous linux)
goweeb --scan-dir ~/Documents/Mangas "one piece"

# Pareil mais sur Windows
goweeb --scan-dir C:\Users\<nom-de-ton-utilisateur>\Documents\Mangas "one piece"
```

#### Spécifier un domaine personnalisé
```bash
# Si le domaine anime-sama change
goweeb -u https://anime-sama.fr --all "one piece"
```

#### Télécharger une structure compatible avec une liseuse

> [!WARNING]
> Il ne s'agit pas de télécharger le manga dans un format EPUB ou autre, mais plutôt de le télécharger
> en suivant une structure particulière utilisée par des logiciels tiers pour formatter les manga à la lecture sur liseuse.

Par exemple, ce mode est très utile pour convertir le manga téléchargé avec [Kindle Comic Converter](https://github.com/ciromattia/kcc)
et ensuite l'upload sur une kindle, kobo ou autre e-reader.

```bash
goweeb -u https://anime-sama.fr --all --ebook-friendly "one piece"
```

#### Combinaison d'options
```bash
# Télécharge les chapitres 1 à 50 de One Piece, dans le dossier "scans" sur Windows 
# en spécifiant un domaine anime-sama personnalisé
goweeb -d 'C:\Users\<nom-de-ton-utilisateur>\scans' -u https://anime-sama.tv "one piece"
```

### Conseils

> [!TIP]
> - La recherche utilise directement le moteur de recherche d'anime-sama, tu peux donc utiliser n'importe quel nom (français, anglais, japonais)
> - Si plusieurs versions d'un scan existent (ex: couleur/noir et blanc), l'outil te demandera laquelle télécharger
> - N'oublie pas les guillemets autour du nom du manga !

> [!WARNING] 
> Garde bien en tête que le domaine d'anime-sama change pour des raisons évidentes... donc n'hésite pas à vérifier [anime-sama.pw](https://anime-sama.pw) pour vérifier quel domaine est actif. Tu peux ensuite le spécifier avec l'argument `-u` comme présenté ci-dessus.

## Crédits

Merci beaucoup à Anime Sama et leur travail colossal sans qui ce projet ne pourrait exister <3
