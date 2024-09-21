package uploads

import (
	"Portfolio/internal/validator"
	"Portfolio/ui"
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// File type with DirEntry embedded and the icon URL
type File struct {
	os.DirEntry
	Icon string
	Path string
	Type string
}

type icon struct {
	Name string
	Path string
}

type Browser struct {
	Dirname string
	Files   []File
}

var functions = template.FuncMap{
	"filename": filename,
	"isDir":    isDir,
}

func filename(file File) string {
	return file.Name()
}

func isDir(file File) bool {
	return file.IsDir()
}

var (
	ErrForbiddenDirectory = errors.New("invalid or forbidden directory")
	ErrFileNotFound       = errors.New("file not found")
	ErrEmptyFileName      = errors.New("empty file name")
	ErrFileName           = errors.New("invalid file name")

	// dirList contains all the allowed directories to list and access from the web app
	dirList = []string{dirs.Root, dirs.Image, dirs.PDF}

	// appExt contains all known application extensions
	appExt = []string{".exe", ".apk", ".app", ".class", ".dll", ".jar", ".war", ".obj"}

	// archiveExt contains all known archive extensions
	archiveExt = []string{".zip", ".7z", ".gz", ".rar", ".deb", ".dmg", ".tar", ".gzip"}

	// audioExt contains all known audio extensions
	audioExt = []string{".wav", ".mp1", ".mp2", ".mp3", ".aiff", ".flac", ".ac3", ".aac", ".ots", ".ogg"}

	// codeExt contains all known source code extensions
	codeExt = []string{".css", ".sass", ".scss", ".less", ".html", ".htm", ".htmx", ".js", ".go", ".py", ".java", ".php", ".sql", ".cpp", ".cxx", ".cc", ".h", ".hpp", ".hxx", ".cs", ".tmpl", ".bat", ".dart", ".fs", ".kt", ".lua", ".sh", "ps1", ".nuc", ".nud", ".pl", ".r", ".rb", ".rs", ".ts", ".vbs", ".json", ".yaml"}

	// docExt contains all known text document extensions
	docExt = []string{".txt", ".docx", ".doc", ".odt", ".csv", ".epub", ".gdoc", ".pages", ".rtf", ".md"}

	// imgExt contains all known image extensions
	imgExt = []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico", ".bmp", ".webm", ".svg", ".odg"}

	// pdfExt contains all known pdf/xps extensions
	pdfExt = []string{".pdf", ".xps"}

	// presentationExt contains all known presentation extensions
	presentationExt = []string{".ppt", ".pptx", ".odp", ".gslides", ".key", ".keynote", ".pez", ".prdx", ".shw", ".pps"}

	// spreadsheetExt contains all known spreadsheet extensions
	spreadsheetExt = []string{".xlsx", ".xls", ".ods", ".gsheet", ".numbers"}

	// videoExt contains all known video extensions
	videoExt = []string{".mp4", ".mpeg", ".mpg", ".mpe", ".mov", ".avi", ".wmv", ".flv", ".ogt", ".wma", ".m4v", ".mkv", ".3gp"}
)

// fileType is the enum-like containing all the icons for any type of file
var fileType = struct {
	Application  icon
	Archive      icon
	Audio        icon
	Code         icon
	Directory    icon
	Document     icon
	File         icon
	Image        icon
	PDF          icon
	Presentation icon
	Spreadsheet  icon
	Video        icon
}{
	Application:  icon{"application", "/static/img/icons/files/application-icon.svg"},
	Archive:      icon{"archive", "/static/img/icons/files/archive-icon.svg"},
	Audio:        icon{"audio", "/static/img/icons/files/audio-icon.svg"},
	Code:         icon{"code", "/static/img/icons/files/code-icon.svg"},
	Directory:    icon{"directory", "/static/img/icons/files/folder-grey-icon.svg"},
	Document:     icon{"document", "/static/img/icons/files/document-icon.svg"},
	File:         icon{"file", "/static/img/icons/files/file-icon.svg"},
	Image:        icon{"image", "/static/img/icons/files/image-icon.svg"},
	PDF:          icon{"pdf", "/static/img/icons/files/pdf-icon.svg"},
	Presentation: icon{"presentation", "/static/img/icons/files/presentation-icon.svg"},
	Spreadsheet:  icon{"spreadsheet", "/static/img/icons/files/spreadsheet-icon.svg"},
	Video:        icon{"video", "/static/img/icons/files/video-icon.svg"},
}

// dirs is the enum-like containing all directories
//
// WHEN ADDING NEW VALUES, ADD THEM TO dirList VARIABLE TOO!
var dirs = struct {
	Root  string
	Image string
	PDF   string
}{
	Root:  "uploads",
	Image: "uploads/img",
	PDF:   "uploads/docs",
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// initDir creates a directory if it doesn't exist
func initDir(dirname string) error {
	if _, err := os.Stat(dirname); os.IsNotExist(err) {
		err = os.MkdirAll(dirname, os.ModePerm)
		return err
	}
	return nil
}

// Init initializes the uploads directory and subdirectories
func Init() error {
	for _, dir := range dirList {
		err := initDir(dir)
		if err != nil {
			return err
		}
	}
	return nil
}

// Add uploads a file in the uploads directory
func Add(file multipart.File, header *multipart.FileHeader) (string, error) {

	// extracting the file name and extension
	_, filename := path.Split(header.Filename)
	ext := path.Ext(filename)
	filename = strings.TrimSuffix(filename, ext)
	if !validator.CheckFileName(filename) {
		filename = "file"
	}

	// setting the appropriate directory according to the file extension
	var dir string
	switch {
	case validator.PermittedValue(ext, imgExt...):
		dir = dirs.Image
	case validator.PermittedValue(ext, pdfExt...):
		dir = dirs.PDF
	default:
		dir = dirs.Root
	}

	// creating the destination file
	filename = filepath.Join(dir, fmt.Sprint(filename, "_", time.Now().Format("2006-01-02T15:04"), ext))
	dst, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("error creating file: %w", err)
	}
	defer dst.Close()

	// upload the file to destination path
	nbBytes, err := io.Copy(dst, file)
	if err != nil {
		return "", fmt.Errorf("error copying file: %w", err)
	}

	// return the message
	return fmt.Sprintf("%d bytes copied to %s", nbBytes, filename), nil
}

// Remove deletes a file
func Remove(file string) error {

	// cleaning the file path
	file = filepath.Clean(file)

	// checking the directory
	if !validator.PermittedValue(filepath.Dir(file), dirList...) {
		return ErrForbiddenDirectory
	}

	// checking the filename format
	if !validator.CheckFileName(filepath.Base(file)) {
		return ErrFileName
	}

	// checking if the file exists
	if !fileExists(file) {
		return ErrFileNotFound
	}

	err := os.Remove(file)
	if err != nil {
		return fmt.Errorf("error removing file: %w", err)
	}

	return nil
}

// Get lists the files and directories in a directory
func Get(dirname string) ([]File, error) {

	// checking if dirname corresponds to an allowed directory
	if !validator.PermittedValue(dirname, dirList...) {
		return nil, ErrForbiddenDirectory
	}

	// creating the File list
	var files []File

	// reading the directory asked for
	entries, err := os.ReadDir(dirname)
	if err != nil {
		return nil, err
	}

	// converting the DirEntries to File types (adding the icon)
	for _, entry := range entries {

		// creating the file with the DirEntry
		var file File
		file.DirEntry = entry
		file.addIconType()
		file.Path = path.Join(dirname, entry.Name())

		// appending the file to the list
		files = append(files, file)
	}

	// returning the files/directories
	return files, nil
}

// addIconType assigns an icon and a file Type to a File according to its extension
func (file *File) addIconType() {

	// checking if the file is a directory
	if file.IsDir() {
		file.Icon = fileType.Directory.Path
		file.Type = fileType.Directory.Name
		return
	}

	// checking the file extension
	ext := strings.ToLower(filepath.Ext(file.Name()))
	switch {

	// image formats
	case validator.PermittedValue(ext, imgExt...):
		file.Icon = fileType.Image.Path
		file.Type = fileType.Image.Name

	// PDF/XPS formats
	case validator.PermittedValue(ext, pdfExt...):
		file.Icon = fileType.PDF.Path
		file.Type = fileType.PDF.Name

	// document formats
	case validator.PermittedValue(ext, docExt...):
		file.Icon = fileType.Document.Path
		file.Type = fileType.Document.Name

	// spreadsheet formats
	case validator.PermittedValue(ext, spreadsheetExt...):
		file.Icon = fileType.Spreadsheet.Path
		file.Type = fileType.Spreadsheet.Name

	// presentation formats
	case validator.PermittedValue(ext, presentationExt...):
		file.Icon = fileType.Presentation.Path
		file.Type = fileType.Presentation.Name

	// video formats
	case validator.PermittedValue(ext, videoExt...):
		file.Icon = fileType.Video.Path
		file.Type = fileType.Video.Name

	// audio formats
	case validator.PermittedValue(ext, audioExt...):
		file.Icon = fileType.Audio.Path
		file.Type = fileType.Audio.Name

	// archive formats
	case validator.PermittedValue(ext, archiveExt...):
		file.Icon = fileType.Archive.Path
		file.Type = fileType.Archive.Name

	// application formats
	case validator.PermittedValue(ext, appExt...):
		file.Icon = fileType.Application.Path
		file.Type = fileType.Application.Name

	// source code formats
	case validator.PermittedValue(ext, codeExt...):
		file.Icon = fileType.Code.Path
		file.Type = fileType.Code.Name

	// other formats
	default:
		file.Icon = fileType.File.Path
		file.Type = fileType.File.Name
	}
}

func Render(w http.ResponseWriter, browser Browser) error {
	tmpl, err := template.New("file-browser").Funcs(functions).ParseFS(ui.Files, "templates/partials/file-browser.tmpl")
	if err != nil {
		return err
	}

	html := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(html, "file-browser", browser)
	if err != nil {
		return err
	}

	// set the response type to text/html
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// if it's all okay, write the status in the header and write the buffer in the ResponseWriter
	w.WriteHeader(http.StatusOK)

	_, err = html.WriteTo(w)
	return err
}
