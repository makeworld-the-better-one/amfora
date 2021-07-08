package display

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	"github.com/makeworld-the-better-one/amfora/config"
	"github.com/makeworld-the-better-one/amfora/structs"
	"github.com/makeworld-the-better-one/amfora/sysopen"
	"github.com/makeworld-the-better-one/go-gemini"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/viper"
)

// For choosing between download and the portal - copy of YesNo basically
var dlChoiceModal = cview.NewModal()

// Channel to indicate what choice they made using the button text
var dlChoiceCh = make(chan string)

var dlModal = cview.NewModal()

func dlInit() {
	panels.AddPanel("dl", dlModal, false, false)
	panels.AddPanel("dlChoice", dlChoiceModal, false, false)

	dlm := dlModal
	chm := dlChoiceModal
	if viper.GetBool("a-general.color") {
		chm.SetButtonBackgroundColor(config.GetColor("btn_bg"))
		chm.SetButtonTextColor(config.GetColor("btn_text"))
		chm.SetBackgroundColor(config.GetColor("dl_choice_modal_bg"))
		chm.SetTextColor(config.GetColor("dl_choice_modal_text"))
		form := chm.GetForm()
		form.SetButtonBackgroundColorFocused(config.GetColor("btn_text"))
		form.SetButtonTextColorFocused(config.GetTextColor("btn_bg", "btn_text"))
		frame := chm.GetFrame()
		frame.SetBorderColor(config.GetColor("dl_choice_modal_text"))
		frame.SetTitleColor(config.GetColor("dl_choice_modal_text"))

		dlm.SetButtonBackgroundColor(config.GetColor("btn_bg"))
		dlm.SetButtonTextColor(config.GetColor("btn_text"))
		dlm.SetBackgroundColor(config.GetColor("dl_modal_bg"))
		dlm.SetTextColor(config.GetColor("dl_modal_text"))
		form = dlm.GetForm()
		form.SetButtonBackgroundColorFocused(config.GetColor("btn_text"))
		form.SetButtonTextColorFocused(config.GetTextColor("btn_bg", "btn_text"))
		frame = dlm.GetFrame()
		frame.SetBorderColor(config.GetColor("dl_modal_text"))
		frame.SetTitleColor(config.GetColor("dl_modal_text"))
	} else {
		chm.SetButtonBackgroundColor(tcell.ColorWhite)
		chm.SetButtonTextColor(tcell.ColorBlack)
		chm.SetBackgroundColor(tcell.ColorBlack)
		chm.SetTextColor(tcell.ColorWhite)
		chm.SetBorderColor(tcell.ColorWhite)
		chm.GetFrame().SetTitleColor(tcell.ColorWhite)
		form := chm.GetForm()
		form.SetButtonBackgroundColorFocused(tcell.ColorBlack)
		form.SetButtonTextColorFocused(tcell.ColorWhite)

		dlm.SetButtonBackgroundColor(tcell.ColorWhite)
		dlm.SetButtonTextColor(tcell.ColorBlack)
		dlm.SetBackgroundColor(tcell.ColorBlack)
		dlm.SetTextColor(tcell.ColorWhite)
		form = dlm.GetForm()
		form.SetButtonBackgroundColorFocused(tcell.ColorBlack)
		form.SetButtonTextColorFocused(tcell.ColorWhite)
		frame := dlm.GetFrame()
		frame.SetBorderColor(tcell.ColorWhite)
		frame.SetTitleColor(tcell.ColorWhite)
	}

	chm.AddButtons([]string{"Open", "Download", "Cancel"})
	chm.SetBorder(true)
	chm.GetFrame().SetTitleAlign(cview.AlignCenter)
	chm.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		dlChoiceCh <- buttonLabel
	})

	dlm.SetBorder(true)
	frame := dlm.GetFrame()
	frame.SetTitleAlign(cview.AlignCenter)
	frame.SetTitle(" Download ")
	dlm.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if buttonLabel == "Ok" {
			panels.HidePanel("dl")
			App.SetFocus(tabs[curTab].view)
			App.Draw()
		}
	})
}

func getMediaHandler(resp *gemini.Response) config.MediaHandler {
	def := config.MediaHandler{
		Cmd:      nil,
		NoPrompt: false,
		Stream:   false,
	}

	mediatype, _, err := mime.ParseMediaType(resp.Meta)
	if err != nil {
		return def
	}

	if ret, ok := config.MediaHandlers[mediatype]; ok {
		return ret
	}

	splitType := strings.Split(mediatype, "/")[0]
	if ret, ok := config.MediaHandlers[splitType]; ok {
		return ret
	}

	if ret, ok := config.MediaHandlers["*"]; ok {
		return ret
	}

	return def
}

// dlChoice displays the download choice modal and acts on the user's choice.
// It should run in a goroutine.
func dlChoice(text, u string, resp *gemini.Response) {
	mediaHandler := getMediaHandler(resp)
	var choice string

	if mediaHandler.NoPrompt {
		choice = "Open"
	} else {
		dlChoiceModal.SetText(text)
		panels.ShowPanel("dlChoice")
		panels.SendToFront("dlChoice")
		App.SetFocus(dlChoiceModal)
		App.Draw()
		choice = <-dlChoiceCh
	}

	if choice == "Download" {
		panels.HidePanel("dlChoice")
		App.Draw()
		downloadURL(config.DownloadsDir, u, resp)
		resp.Body.Close() // Only close when the file is downloaded
		return
	}
	if choice == "Open" {
		panels.HidePanel("dlChoice")
		App.Draw()
		open(u, resp)
		return
	}

	// They chose the "Cancel" button
	panels.HidePanel("dlChoice")
	App.SetFocus(tabs[curTab].view)
	App.Draw()
}

// open performs the same actions as downloadURL except it also opens the file.
// If there is no system viewer configured for the particular mediatype, it opens it
// with the default system viewer.
func open(u string, resp *gemini.Response) {
	mediaHandler := getMediaHandler(resp)

	if mediaHandler.Stream {
		// Run command with downloaded data from stdin

		cmd := mediaHandler.Cmd
		var proc *exec.Cmd
		if len(cmd) == 1 {
			proc = exec.Command(cmd[0])
		} else {
			proc = exec.Command(cmd[0], cmd[1:]...)
		}
		proc.Stdin = resp.Body

		err := proc.Start()
		if err != nil {
			Error("File Opening Error", "Error executing custom command: "+err.Error())
			return
		}
		Info("Opened with " + cmd[0])
		return
	}

	path := downloadURL(config.TempDownloadsDir, u, resp)
	if path == "" {
		return
	}

	panels.HidePanel("dl")
	App.SetFocus(tabs[curTab].view)
	App.Draw()

	if mediaHandler.Cmd == nil {
		// Open with system default viewer
		_, err := sysopen.Open(path)
		if err != nil {
			Error("System Viewer Error", err.Error())
			return
		}
		Info("Opened in default system viewer")
	} else {
		cmd := mediaHandler.Cmd
		err := exec.Command(cmd[0], append(cmd[1:], path)...).Start()
		if err != nil {
			Error("File Opening Error", "Error executing custom command: "+err.Error())
			return
		}
		Info("Opened with " + cmd[0])
	}
	App.Draw()
}

// downloadURL pulls up a modal to show download progress and saves the URL content.
// downloadPage should be used for Page content.
// Returns location downloaded to or an empty string on error.
func downloadURL(dir, u string, resp *gemini.Response) string {
	_, _, width, _ := dlModal.GetInnerRect()
	// Copy of progressbar.DefaultBytesSilent with custom width
	bar := progressbar.NewOptions64(
		-1,
		progressbar.OptionSetWidth(width),
		progressbar.OptionSetWriter(ioutil.Discard),
		progressbar.OptionShowBytes(true),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionSpinnerType(14),
	)
	bar.RenderBlank() //nolint:errcheck

	savePath, err := downloadNameFromURL(dir, u, "")
	if err != nil {
		Error("Download Error", "Error deciding on file name: "+err.Error())
		return ""
	}
	f, err := os.OpenFile(savePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		Error("Download Error", "Error creating download file: "+err.Error())
		return ""
	}
	defer f.Close()

	done := false

	go func(isDone *bool) {
		// Update the bar display
		for !*isDone {
			dlModal.SetText(bar.String())
			App.Draw()
			time.Sleep(100 * time.Millisecond)
		}
	}(&done)

	// Display
	dlModal.ClearButtons()
	dlModal.AddButtons([]string{"Downloading..."})
	panels.ShowPanel("dl")
	panels.SendToFront("dl")
	App.SetFocus(dlModal)
	App.Draw()

	_, err = io.Copy(io.MultiWriter(f, bar), resp.Body)
	done = true
	if err != nil {
		panels.HidePanel("dl")
		Error("Download Error", err.Error())
		f.Close()
		os.Remove(savePath) // Remove partial file
		return ""
	}
	dlModal.SetText(fmt.Sprintf("Download complete! File saved to %s.", savePath))
	dlModal.ClearButtons()
	dlModal.AddButtons([]string{"Ok"})
	dlModal.GetForm().SetFocus(100)
	App.SetFocus(dlModal)
	App.Draw()

	return savePath
}

// downloadPage saves the passed Page to a file.
// It returns the saved path and an error.
// It always cleans up, so if an error is returned there is no file saved
func downloadPage(p *structs.Page) (string, error) {
	var savePath string
	var err error

	if p.Mediatype == structs.TextGemini {
		savePath, err = downloadNameFromURL(config.DownloadsDir, p.URL, ".gmi")
	} else {
		savePath, err = downloadNameFromURL(config.DownloadsDir, p.URL, ".txt")
	}
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(savePath, []byte(p.Raw), 0644)
	if err != nil {
		// Just in case
		os.Remove(savePath)
		return "", err
	}
	return savePath, err
}

// downloadNameFromURL takes a URl and returns a safe download path that will not overwrite any existing file.
// ext is an extension that will be added if the file has no extension, and for domain only URLs.
// It should include the dot.
func downloadNameFromURL(dir, u, ext string) (string, error) {
	var name string
	var err error

	parsed, _ := url.Parse(u)
	if strings.HasPrefix(u, "about:") {
		name, err = getSafeDownloadName(dir, parsed.Opaque+ext, true, 0)
		if err != nil {
			return "", err
		}
	} else if parsed.Path == "" || path.Base(parsed.Path) == "/" {
		// No file, just the root domain
		name, err = getSafeDownloadName(dir, parsed.Hostname()+ext, true, 0)
		if err != nil {
			return "", err
		}
	} else {
		// There's a specific file
		name = path.Base(parsed.Path)
		if !strings.Contains(name, ".") {
			// No extension
			name += ext
		}
		name, err = getSafeDownloadName(dir, name, false, 0)
		if err != nil {
			return "", err
		}
	}

	return filepath.Join(dir, name), nil
}

// getSafeDownloadName is used by downloads.go only.
// It returns a modified name that is unique for the specified folder.
// This way duplicate saved files will not overwrite each other.
//
// lastDot should be set to true if the number added to the name should come before
// the last dot in the filename instead of the first.
//
// n should be set to 0, it is used for recursiveness.
func getSafeDownloadName(dir, name string, lastDot bool, n int) (string, error) {
	// newName("test.txt", 3) -> "test(3).txt"
	newName := func() string {
		if n <= 0 {
			return name
		}
		if lastDot {
			ext := filepath.Ext(name)
			return strings.TrimSuffix(name, ext) + "(" + strconv.Itoa(n) + ")" + ext
		}
		idx := strings.Index(name, ".")
		if idx == -1 {
			return name + "(" + strconv.Itoa(n) + ")"
		}
		return name[:idx] + "(" + strconv.Itoa(n) + ")" + name[idx:]
	}

	d, err := os.Open(dir)
	if err != nil {
		return "", err
	}
	files, err := d.Readdirnames(-1)
	if err != nil {
		d.Close()
		return "", err
	}

	nn := newName()
	for i := range files {
		if nn == files[i] {
			d.Close()
			return getSafeDownloadName(dir, name, lastDot, n+1)
		}
	}
	d.Close()
	return nn, nil // Name doesn't exist already
}
