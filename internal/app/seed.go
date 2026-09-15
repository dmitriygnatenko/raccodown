package app

import (
	"context"
	"fmt"
	"log/slog"

	"raccodown/internal/domain/usecase"
	"raccodown/internal/port"
)

// seedDemoNotes inserts one welcome note per supported UI language (ru, en, es, de, fr — see the
// frontend's SUPPORTED_LANGUAGES) the first time the repository is empty, so a fresh checkout has
// something to look at. Each note demonstrates every formatting tool on the editor's toolbar (see
// web/js/data/markdown-commands.js): heading, bold, italic, strikethrough, blockquote, bullet and
// numbered lists, link, code block, table, and horizontal rule. It's a no-op once any note exists.
func seedDemoNotes(ctx context.Context, repo port.NoteRepository) error {
	existing, err := repo.List(ctx, port.NoteListFilter{})
	if err != nil {
		return err
	}

	if len(existing) > 0 {
		return nil
	}

	seed := []port.NoteCreateRequest{
		{
			Title: "Добро пожаловать в Raccodown",
			Content: "# Добро пожаловать 🦝\n\n## Панель инструментов\n\n" +
				"**Raccodown** — self-hosted заметки в *Markdown*, ~~а не в блокноте~~.\n\n" +
				"> Цитата — кнопка ❝ на панели\n\n" +
				"- Маркированный список\n- Ещё один пункт\n\n" +
				"1. Нумерованный список\n2. Второй пункт\n\n" +
				"[Ссылка](https://daringfireball.net/projects/markdown/) — кнопка 🔗\n\n" +
				"```go\nfunc main() {}\n```\n\n" +
				"| Заголовок 1 | Заголовок 2 |\n| --- | --- |\n| Ячейка | Ячейка |\n\n" +
				"---\n\n#raccodown #welcome",
		},
		{
			Title: "Welcome to Raccodown",
			Content: "# Welcome 🦝\n\n## The toolbar\n\n" +
				"**Raccodown** is a self-hosted *Markdown* notes app, ~~not a plain text box~~.\n\n" +
				"> A blockquote — the ❝ button\n\n" +
				"- A bullet list\n- Another item\n\n" +
				"1. A numbered list\n2. Second item\n\n" +
				"[A link](https://daringfireball.net/projects/markdown/) — the 🔗 button\n\n" +
				"```go\nfunc main() {}\n```\n\n" +
				"| Header 1 | Header 2 |\n| --- | --- |\n| Cell | Cell |\n\n" +
				"---\n\n#raccodown #welcome",
		},
		{
			Title: "Bienvenido a Raccodown",
			Content: "# Bienvenido 🦝\n\n## La barra de herramientas\n\n" +
				"**Raccodown** son notas en *Markdown* autoalojadas, ~~no un bloc de notas~~.\n\n" +
				"> Una cita — el botón ❝\n\n" +
				"- Una lista con viñetas\n- Otro elemento\n\n" +
				"1. Una lista numerada\n2. Segundo elemento\n\n" +
				"[Un enlace](https://daringfireball.net/projects/markdown/) — el botón 🔗\n\n" +
				"```go\nfunc main() {}\n```\n\n" +
				"| Encabezado 1 | Encabezado 2 |\n| --- | --- |\n| Celda | Celda |\n\n" +
				"---\n\n#raccodown #welcome",
		},
		{
			Title: "Willkommen bei Raccodown",
			Content: "# Willkommen 🦝\n\n## Die Werkzeugleiste\n\n" +
				"**Raccodown** sind selbst gehostete *Markdown*-Notizen, ~~kein einfaches Textfeld~~.\n\n" +
				"> Ein Zitat — die ❝-Schaltfläche\n\n" +
				"- Eine Aufzählungsliste\n- Noch ein Punkt\n\n" +
				"1. Eine nummerierte Liste\n2. Zweiter Punkt\n\n" +
				"[Ein Link](https://daringfireball.net/projects/markdown/) — die 🔗-Schaltfläche\n\n" +
				"```go\nfunc main() {}\n```\n\n" +
				"| Überschrift 1 | Überschrift 2 |\n| --- | --- |\n| Zelle | Zelle |\n\n" +
				"---\n\n#raccodown #welcome",
		},
		{
			Title: "Bienvenue sur Raccodown",
			Content: "# Bienvenue 🦝\n\n## La barre d'outils\n\n" +
				"**Raccodown** est une appli de notes *Markdown* auto-hébergée, ~~pas un bloc-notes~~.\n\n" +
				"> Une citation — le bouton ❝\n\n" +
				"- Une liste à puces\n- Un autre élément\n\n" +
				"1. Une liste numérotée\n2. Deuxième élément\n\n" +
				"[Un lien](https://daringfireball.net/projects/markdown/) — le bouton 🔗\n\n" +
				"```go\nfunc main() {}\n```\n\n" +
				"| Titre 1 | Titre 2 |\n| --- | --- |\n| Cellule | Cellule |\n\n" +
				"---\n\n#raccodown #welcome",
		},
	}

	for _, note := range seed {
		note.Tags = usecase.ExtractTags(note.Content)

		if _, err = repo.Create(ctx, note); err != nil {
			return err
		}
	}

	return nil
}

// seedDemoUserParams bundles the dependencies and demo credentials seedDemoUser needs.
type seedDemoUserParams struct {
	Users    port.UserRepository
	Hasher   port.PasswordHasher
	Username string
	Password string
}

// seedDemoUser creates the single account the first time it's ever called against an empty user
// repository — raccodown is single-user, so this is what makes a fresh checkout loggable-into
// without a registration flow. It's a no-op once any user exists.
func seedDemoUser(ctx context.Context, params seedDemoUserParams) error {
	count, err := params.Users.Count(ctx)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	hash, err := params.Hasher.Hash(params.Password)
	if err != nil {
		return err
	}

	if _, err = params.Users.Create(ctx, port.UserCreateRequest{
		Username:     usecase.NormalizeUsername(params.Username),
		PasswordHash: hash,
	}); err != nil {
		return fmt.Errorf("create demo user: %w", err)
	}

	slog.InfoContext(ctx, "seeded the demo account", "username", params.Username)

	return nil
}
