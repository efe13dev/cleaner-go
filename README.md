# Cleaner

**Cleaner** es una utilidad de terminal escrita en Go para encontrar y eliminar archivos, caches y artefactos de desarrollo que ocupan espacio innecesario.

Incluye una interfaz interactiva basada en Terminal UI y permite revisar cada elemento antes de eliminarlo.

## Caracteristicas

- Detecta caches temporales y del sistema.
- Encuentra artefactos de desarrollo como `node_modules`, `.venv`, `dist`, `build`, `target` y `.next`.
- Identifica archivos individuales mayores o iguales a 100 MiB.
- Funciona de forma interactiva desde la terminal.
- Permite seleccionar varios elementos antes de confirmar el borrado.
- Protege archivos especiales como discos virtuales (`.vhdx`, `.vmdk`, `.qcow2`, etc.).
- Conserva `desktop.ini` al limpiar directorios de cache.
- Ordena los resultados por categoria y tamaño.

## Categorias

| Categoria | Descripcion |
|---|---|
| **Cache** | Archivos temporales o caches del sistema que pueden regenerarse |
| **Desarrollo** | Dependencias y artefactos generados por herramientas de desarrollo |
| **Archivos grandes** | Archivos individuales de al menos 100 MiB que requieren revision manual |

Los archivos grandes no se seleccionan automaticamente con la opcion de limpieza rapida.

## Requisitos

- Go compatible con el modulo definido en `go.mod`.
- Una terminal compatible con interfaces TUI.

## Ejecucion

Desde la carpeta del proyecto:

```bash
go run .
```

Por defecto, Cleaner toma como raiz el directorio del usuario y tambien analiza las rutas conocidas de caches del sistema.

Tambien puedes indicar una carpeta concreta:

```bash
go run . /ruta/a/analizar
```

## Controles

| Tecla | Accion |
|---|---|
| `↑` / `k` | Moverse hacia arriba |
| `↓` / `j` | Moverse hacia abajo |
| `Espacio` | Seleccionar o quitar seleccion |
| `a` | Seleccionar todos los elementos seguros |
| `Enter` / `d` | Solicitar confirmacion de borrado |
| `y` | Confirmar el borrado |
| `q` / `Ctrl+C` | Salir |

## Seguridad

Cleaner solicita confirmacion antes de borrar elementos seleccionados.

Antes de confirmar, ten especial cuidado con:

- Archivos detectados en la categoria de archivos grandes.
- Carpetas personales que puedan contener datos importantes.
- Respaldos, archivos multimedia, documentos o discos virtuales.
- Directorios marcados como artefactos si contienen datos modificados manualmente.

El programa excluye de forma preventiva directorios y extensiones sensibles como Windows, `AppData`, repositorios `.git`, archivos virtuales y otros recursos protegidos.

## Compilar

Para crear un ejecutable:

```bash
go build -o cleaner .
```

Para ejecutar las pruebas:

```bash
go test ./...
```

## Dependencias

- [Bubble Tea](https://github.com/charmbracelet/bubbletea): interfaz interactiva de terminal.
- [Lip Gloss](https://github.com/charmbracelet/lipgloss): estilos y presentacion visual.

## Licencia

Agrega una licencia antes de distribuir el proyecto si deseas definir claramente como puede utilizarse, modificarse y redistribuirse.
