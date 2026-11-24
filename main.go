package main

import (
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	subStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	freedStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	warnStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	cursorStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	checkedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	sizeStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))

	catOrder = map[string]int{"cache": 0, "dev": 1, "large": 2}
	catTitle = map[string]string{
		"cache": "CACHES DEL SISTEMA (seguro)",
		"dev":   "ARTEFACTOS DEV (seguro)",
		"large": "ARCHIVOS GRANDES >= 100 MiB (revisar manual)",
	}
	catStyle = map[string]lipgloss.Style{
		"cache": lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("45")),
		"dev":   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")),
		"large": lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")),
	}
)

const largeFileThreshold = 100 << 20 // 100 MiB

var devArtifacts = map[string]string{
	"node_modules": "Dependencias de Node.js (se regenera con npm/pnpm/bun install)",
	".next":        "Cache de compilacion de Next.js (se regenera con next build/dev)",
	"dist":         "Carpeta de salida compilada del proyecto (se regenera al compilar)",
	"build":        "Carpeta de compilacion del proyecto (se regenera al compilar)",
	"target":       "Artefactos compilados de Rust/Maven (se regenera con cargo/mvn build)",
	".venv":        "Entorno virtual de Python (se regenera con venv/uv/pip)",
	"venv":         "Entorno virtual de Python (se regenera con venv/uv/pip)",
	".turbo":       "Cache de compilacion de Turborepo",
	".gradle":      "Cache local de compilacion de Gradle",
}

var skipWalkDirs = map[string]bool{
	".git":                      true,
	"AppData":                   true,
	"Windows":                   true,
	"Program Files":             true,
	"Program Files (x86)":       true,
	"ProgramData":               true,
	"$Recycle.Bin":              true,
	"System Volume Information": true,
	"Recovery":                  true,
	".vscode":                   true,
	".cursor":                   true,
	".windsurf":                 true,
	".cargo":                    true,
	".rustup":                   true,
	".local":                    true,
	".cache":                    true,
	".npm":                      true,
	".bun":                      true,
}

var ambiguousDevDirs = map[string]bool{
	"dist":   true,
	"build":  true,
	"target": true,
	"venv":   true,
}

var projectMarkers = []string{
	"package.json", "Cargo.toml", "pyproject.toml", "requirements.txt", "go.mod", "pom.xml", "build.gradle", ".git",
}

var protectedExts = map[string]bool{
	".vhdx": true, ".vhd": true, ".vmdk": true, ".vdi": true, ".qcow2": true,
}

func isProjectDir(dir string) bool {
	for _, m := range projectMarkers {
		if _, err := os.Lstat(filepath.Join(dir, m)); err == nil {
			return true
		}
	}
	return false
}

type Item struct {
	Path     string
	Size     int64
	Category string
	Desc     string
	Selected bool
}

func dirSize(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == "desktop.ini" || d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func cleanItem(it Item) (freed int64, remaining int64) {
	if it.Category == "cache" {
		if entries, err := os.ReadDir(it.Path); err == nil {
			for _, e := range entries {
				if e.Name() != "desktop.ini" {
					_ = os.RemoveAll(filepath.Join(it.Path, e.Name()))
				}
			}
		}
	} else if err := os.RemoveAll(it.Path); err == nil {
		return it.Size, 0
	}
	rem := dirSize(it.Path)
	if info, err := os.Lstat(it.Path); err == nil && !info.IsDir() {
		rem = info.Size()
	}
	if rem < it.Size {
		return it.Size - rem, rem
	}
	return 0, it.Size
}

func scan(root string, includeSystemCaches bool, minFileSize int64) ([]Item, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var items []Item

	add := func(p, cat, desc string, size int64) {
		if size > 0 && !seen[p] {
			seen[p] = true
			items = append(items, Item{Path: p, Size: size, Category: cat, Desc: desc})
		}
	}

	if includeSystemCaches {
		if home, err := os.UserHomeDir(); err == nil {
			for _, c := range []struct{ rel, desc string }{
				{".cache/go-build", "Cache de compilacion de Go"},
				{".cache/pip", "Paquetes descargados de Python (pip)"},
				{".cache/uv", "Paquetes descargados de Python (uv)"},
				{".cache/pnpm", "Cache de paquetes de pnpm"},
				{".cache/yarn", "Cache de paquetes de Yarn"},
				{".cache/typescript", "Cache de analisis de TypeScript"},
				{".cache/node-gyp", "Cabeceras C++ para compilar modulos nativos de Node.js"},
				{".npm/_cacache", "Paquetes descargados por npm"},
				{".pnpm-store", "Almacen de paquetes de pnpm"},
				{".local/share/pnpm/store", "Almacen de paquetes de pnpm"},
				{".bun/install/cache", "Paquetes descargados por Bun"},
				{".cargo/registry/cache", "Crates descargados de Rust (Cargo)"},
				{"go/pkg/mod/cache", "Modulos descargados de Go"},
				{".local/share/Trash/files", "Papelera de reciclaje de Linux"},
				{"AppData/Local/Temp", "Archivos temporales de programas e instaladores de Windows"},
				{"AppData/Local/CrashDumps", "Volcados de memoria de programas que se cerraron con error"},
				{"AppData/Local/D3DSCache", "Shaders compilados de DirectX (juegos y apps 3D)"},
				{"AppData/Local/SquirrelTemp", "Temporales de actualizadores (Discord, Teams, Postman, etc.)"},
				{"AppData/Local/cache", "Cache general de herramientas CLI"},
				{"AppData/Local/npm-cache", "Paquetes descargados por npm"},
				{"AppData/Local/pnpm/store", "Almacen de paquetes de pnpm"},
				{"AppData/Local/pip/cache", "Paquetes descargados de Python (pip)"},
				{"AppData/Local/uv/cache", "Paquetes descargados de Python (uv)"},
				{"AppData/Local/NuGet/v3-cache", "Paquetes descargados de .NET (NuGet)"},
				{"AppData/Local/go-build", "Cache de compilacion de Go"},
				{"AppData/Local/node-gyp", "Cabeceras C++ para compilar modulos nativos de Node.js"},
				{"AppData/Local/Microsoft/Windows/INetCache", "Cache web interna de Windows y sus aplicaciones"},
				{"AppData/Local/Microsoft/Edge/User Data/Default/Cache", "Imagenes y archivos web en cache de Microsoft Edge (no borra sesiones)"},
				{"AppData/Local/Microsoft/Edge/User Data/Default/Code Cache", "Codigo JS/WASM compilado de webs en Microsoft Edge (no borra sesiones)"},
				{"AppData/Local/Microsoft/Edge/User Data/Default/GPUCache", "Shaders graficos de Microsoft Edge"},
				{"AppData/Local/Google/Chrome/User Data/Default/Cache", "Imagenes y archivos web en cache de Google Chrome (no borra sesiones)"},
				{"AppData/Local/Google/Chrome/User Data/Default/Code Cache", "Codigo JS/WASM compilado de webs en Google Chrome (no borra sesiones)"},
				{"AppData/Local/Google/Chrome/User Data/Default/GPUCache", "Shaders graficos de Google Chrome"},
				{"AppData/Local/BraveSoftware/Brave-Browser/User Data/Default/Cache", "Imagenes y archivos web en cache de Brave (no borra sesiones)"},
				{"AppData/Local/BraveSoftware/Brave-Browser/User Data/Default/Code Cache", "Codigo JS/WASM compilado de webs en Brave (no borra sesiones)"},
				{"AppData/Local/BraveSoftware/Brave-Browser/User Data/Default/GPUCache", "Shaders graficos de Brave"},
				{"AppData/Local/Discord/Cache", "Imagenes y adjuntos en cache de Discord (no cierra sesion)"},
				{"AppData/Local/Discord/Code Cache", "Codigo JS compilado de Discord"},
				{"AppData/Local/Discord/GPUCache", "Shaders graficos de Discord"},
				{"AppData/Local/NVIDIA/DXCache", "Shaders DirectX compilados por el driver de NVIDIA"},
				{"AppData/Local/NVIDIA/GLCache", "Shaders OpenGL compilados por el driver de NVIDIA"},
				{"AppData/Local/AMD/DxCache", "Shaders DirectX compilados por el driver de AMD"},
				{"Pictures/Screenshots", "Capturas de pantalla guardadas de Windows"},
				{"Videos/Captures", "Grabaciones de pantalla y clips de Xbox Game Bar"},
				{"AppData/Local/Packages/Microsoft.ScreenSketch_8wekyb3d8bbwe/TempState", "Capturas temporales de la herramienta Recortes de Windows"},
			} {
				p := filepath.Join(home, c.rel)
				if info, err := os.Lstat(p); err == nil && info.IsDir() {
					add(p, "cache", c.desc, dirSize(p))
				}
			}
		}
	}

	err = filepath.WalkDir(absRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if seen[p] || (skipWalkDirs[name] && p != absRoot) {
				return filepath.SkipDir
			}
			if desc, ok := devArtifacts[name]; ok && p != absRoot {
				if ambiguousDevDirs[name] && !isProjectDir(filepath.Dir(p)) {
					return filepath.SkipDir
				}
				add(p, "dev", desc, dirSize(p))
				return filepath.SkipDir
			}
			if strings.HasPrefix(name, ".") && p != absRoot {
				return filepath.SkipDir
			}
			return nil
		}
		if protectedExts[strings.ToLower(filepath.Ext(p))] {
			return nil
		}
		if info, err := d.Info(); err == nil && info.Size() >= minFileSize {
			add(p, "large", "Archivo pesado (>= 100 MiB) — revisa que no sea personal antes de borrar", info.Size())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.SortFunc(items, func(a, b Item) int {
		if c := cmp.Compare(catOrder[a.Category], catOrder[b.Category]); c != 0 {
			return c
		}
		return cmp.Compare(b.Size, a.Size)
	})
	return items, nil
}

func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

type scanMsg struct {
	items []Item
	err   error
}

type model struct {
	root       string
	items      []Item
	cursor     int
	height     int
	scanning   bool
	confirming bool
	freed      int64
	deleted    int
	err        error
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		// shortcut: includeSystemCaches always true on interactive run, add CLI flags when needed
		items, err := scan(m.root, true, largeFileThreshold)
		return scanMsg{items: items, err: err}
	}
}

func (m model) selectedStats() (int, int64) {
	var count int
	var bytes int64
	for _, it := range m.items {
		if it.Selected {
			count++
			bytes += it.Size
		}
	}
	return count, bytes
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
	case scanMsg:
		m.scanning = false
		m.items, m.err = msg.items, msg.err
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" || (key == "q" && !m.confirming) {
			return m, tea.Quit
		}
		if m.scanning {
			return m, nil
		}
		if m.confirming {
			switch key {
			case "y", "Y":
				var kept []Item
				for _, it := range m.items {
					if it.Selected {
						freed, rem := cleanItem(it)
						if freed > 0 {
							m.freed += freed
							m.deleted++
						}
						if rem == 0 {
							continue
						}
						it.Size = rem
						it.Selected = false
					}
					kept = append(kept, it)
				}
				m.items = kept
				if m.cursor >= len(m.items) && m.cursor > 0 {
					m.cursor = len(m.items) - 1
				}
				m.confirming = false
			default:
				m.confirming = false
			}
			return m, nil
		}

		switch key {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case " ":
			if len(m.items) > 0 {
				m.items[m.cursor].Selected = !m.items[m.cursor].Selected
			}
		case "a":
			var safeTotal, safeSelected int
			for _, it := range m.items {
				if it.Category != "large" {
					safeTotal++
					if it.Selected {
						safeSelected++
					}
				}
			}
			selectSafe := safeSelected < safeTotal
			for i := range m.items {
				if m.items[i].Category != "large" {
					m.items[i].Selected = selectSafe
				}
			}
		case "enter", "d":
			if count, _ := m.selectedStats(); count > 0 {
				m.confirming = true
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.scanning {
		return titleStyle.Render("CLEANER") + "\n" + subStyle.Render(fmt.Sprintf("Escaneando %s y caches del sistema... (q para salir)", m.root)) + "\n"
	}
	if m.err != nil {
		return warnStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("CLEANER"))
	b.WriteString(subStyle.Render(fmt.Sprintf("  Raiz: %s (%d encontrados)", m.root, len(m.items))))
	b.WriteString("\n")
	if m.deleted > 0 {
		b.WriteString(freedStyle.Render(fmt.Sprintf("Liberado en esta sesion: %s (%d borrados)", humanSize(m.freed), m.deleted)))
		b.WriteString("\n")
	}

	if len(m.items) == 0 {
		b.WriteString("\nNada que limpiar.\n\n")
		b.WriteString(subStyle.Render("[q] salir"))
		b.WriteString("\n")
		return b.String()
	}

	maxRows := 15
	if m.height > 10 {
		maxRows = m.height - 10
	}
	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := min(start+maxRows, len(m.items))

	prevCat := ""
	for i := start; i < end; i++ {
		it := m.items[i]
		if it.Category != prevCat {
			prevCat = it.Category
			b.WriteString("\n")
			b.WriteString(catStyle[it.Category].Render("── " + catTitle[it.Category] + " ──"))
			b.WriteString("\n")
		}
		cur := "  "
		if i == m.cursor {
			cur = cursorStyle.Render("❯ ")
		}
		chk := subStyle.Render("[ ]")
		if it.Selected {
			chk = checkedStyle.Render("[✓]")
		}
		sz := sizeStyle.Render(fmt.Sprintf("%9s", humanSize(it.Size)))
		tag := catStyle[it.Category].Render(fmt.Sprintf("[%-5s]", it.Category))
		pathStr := it.Path
		if i == m.cursor {
			pathStr = cursorStyle.Render(it.Path)
		}
		b.WriteString(fmt.Sprintf("%s%s %s  %s  %s\n", cur, chk, sz, tag, pathStr))
	}

	if m.cursor < len(m.items) && m.items[m.cursor].Desc != "" {
		b.WriteString("\n")
		b.WriteString(titleStyle.Render("ℹ"))
		b.WriteString("  ")
		b.WriteString(subStyle.Render(m.items[m.cursor].Desc))
		b.WriteString("\n")
	}
	count, bytes := m.selectedStats()
	b.WriteString(sizeStyle.Render(fmt.Sprintf("Seleccionado: %d items (%s)", count, humanSize(bytes))))
	b.WriteString("\n")
	if m.confirming {
		b.WriteString(warnStyle.Render(fmt.Sprintf("¿CONFIRMAR BORRADO PERMANENTE de %d items (%s)? [y/N]", count, humanSize(bytes))))
		b.WriteString("\n")
	} else {
		b.WriteString(subStyle.Render("[↑/↓ o j/k] mover  [espacio] marcar  [a] marcar dev+cache  [enter/d] borrar  [q] salir"))
		b.WriteString("\n")
	}
	return b.String()
}

func main() {
	root := "."
	if home, err := os.UserHomeDir(); err == nil {
		root = home
	}
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	p := tea.NewProgram(model{root: root, scanning: true}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
