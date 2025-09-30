package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// AppState holds the application's state, including the main window and UI components
// that need to be accessed globally for updates.
type AppState struct {
	window     fyne.Window
	translator *i18n.Localizer

	packagesLabel    *widget.Label
	listButton       *widget.Button
	destinationLabel *widget.Label
	radioDest        *widget.RadioGroup
	sendButton       *widget.Button
}

// updateUI refreshes the text of all widgets when the language is changed.
func (a *AppState) updateUI() {
	a.window.SetTitle(a.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Title"}))
	a.packagesLabel.SetText(a.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Label.Packages"}))
	a.listButton.SetText(a.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Button.List"}))
	a.destinationLabel.SetText(a.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Label.Destination"}))
	a.radioDest.Options[0] = a.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Dest.Files"})
	a.radioDest.Options[1] = a.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Dest.Obb"})
	a.radioDest.Refresh()
	a.sendButton.SetText(a.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Button.Send"}))
}

// main initializes the application, sets up the UI, and runs the main event loop.
func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("ADB File Pusher")
	myWindow.Resize(fyne.NewSize(1000, 768))

	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	bundle.LoadMessageFile("i18n/en.toml")
	bundle.LoadMessageFile("i18n/es.toml")

	translator := i18n.NewLocalizer(bundle, language.English.String())
	appState := &AppState{window: myWindow, translator: translator}
	myWindow.SetTitle(translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Title"}))

	packages := binding.NewStringList()
	selectedPackage := binding.NewString()
	var allPackages []string // Stores the complete list of packages for filtering

	statusBinding := binding.NewString()
	statusBinding.Set(translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Status.Ready"}))
	statusLabel := widget.NewLabelWithData(statusBinding)

	progressBinding := binding.NewFloat()
	progressVisible := binding.NewBool()
	progressVisible.Set(false)

	progressBar := widget.NewProgressBarWithData(progressBinding)
	progressVisible.AddListener(binding.NewDataListener(func() {
		visible, _ := progressVisible.Get()
		if visible {
			progressBar.Show()
		} else {
			progressBar.Hide()
		}
	}))

	packageList := widget.NewListWithData(packages,
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(i.(binding.String))
		})

	packageList.OnSelected = func(id widget.ListItemID) {
		pkg, _ := packages.GetValue(id)
		selectedPackage.Set(pkg)
	}

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder(translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Search.Packages"})) // Localized placeholder
	searchEntry.OnChanged = func(text string) {
		searchText := strings.TrimSpace(strings.ToLower(text))

		if searchText == "" {
			packages.Set(allPackages)
			return
		}

		searchTerms := strings.Fields(searchText)

		filteredList := []string{}
		for _, pkg := range allPackages {
			pkgLower := strings.ToLower(pkg)
			matchesAll := true
			for _, term := range searchTerms {
				if !strings.Contains(pkgLower, term) {
					matchesAll = false
					break
				}
			}

			if matchesAll {
				filteredList = append(filteredList, pkg)
			}
		}
		packages.Set(filteredList)
	}

	appState.packagesLabel = widget.NewLabel(translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Label.Packages"}))
	appState.listButton = widget.NewButton(translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Button.List"}), func() {
		go listPackages(packages, &allPackages, statusBinding, appState)
	})

	appState.destinationLabel = widget.NewLabel(translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Label.Destination"}))
	destChoice := "files"
	appState.radioDest = widget.NewRadioGroup([]string{
		translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Dest.Files"}),
		translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Dest.Obb"}),
	}, func(selected string) {
		if selected == appState.radioDest.Options[0] {
			destChoice = "files"
		} else {
			destChoice = "obb"
		}
	})
	appState.radioDest.SetSelected(appState.radioDest.Options[0])

	appState.sendButton = widget.NewButton(translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Button.Send"}), func() {
		currentPackage, _ := selectedPackage.Get()
		if currentPackage == "" {
			msg := translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Error.NoPackage"})
			dialog.ShowError(fmt.Errorf(msg), myWindow)
			return
		}
		openFileDialog(currentPackage, destChoice, myWindow, progressBinding, progressVisible, statusBinding, appState)
	})

	langSelector := widget.NewSelect([]string{"English", "Spanish"}, func(lang string) {
		var langTag language.Tag
		if lang == "Spanish" {
			langTag = language.Spanish
		} else {
			langTag = language.English
		}
		appState.translator = i18n.NewLocalizer(bundle, langTag.String())
		appState.updateUI()
	})
	langSelector.SetSelected("English")

	// Layout of the main application window.
	topSection := container.NewBorder(
		container.NewVBox(
			appState.packagesLabel,
			searchEntry,
		), // Top: Label and Search
		appState.listButton, // Bottom: Button
		nil,                 // Left
		nil,                 // Right
		packageList,         // Center: List will expand to fill space
	)

	bottomSection := container.NewVBox(
		appState.destinationLabel,
		appState.radioDest,
		appState.sendButton,
		progressBar,
	)

	split := container.NewVSplit(topSection, bottomSection)
	split.Offset = 0.6 // Give 60% of the space to the top section

	content := container.NewBorder(langSelector, statusLabel, nil, nil, split)
	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}

// listPackages fetches the list of third-party packages from a connected Android device
// using ADB. It runs in a goroutine to avoid blocking the UI.
func listPackages(packages binding.StringList, allPkgs *[]string, status binding.String, state *AppState) {
	status.Set(state.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Status.Listing"}))

	cmd := exec.Command("adb", "shell", "pm", "list", "packages", "-3")
	output, err := cmd.Output()
	if err != nil {
		var msg string
		if strings.Contains(err.Error(), "executable file not found") {
			msg = state.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Error.ADBNotFound"})
		} else {
			msg = state.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Error.DeviceNotFound"})
		}
		status.Set(msg)
		return
	}

	lines := strings.Split(string(output), "\n")
	var pkgList []string
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimPrefix(line, "package:"))
		if line != "" {
			pkgList = append(pkgList, line)
		}
	}
	sort.Strings(pkgList)
	*allPkgs = pkgList      // Update the master list
	packages.Set(pkgList) // Update the visible list

	status.Set(state.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Status.Ready"}))
}

// openFileDialog displays a custom file selection dialog with search and directory navigation.
func openFileDialog(pkg, dest string, win fyne.Window, progress binding.Float, progressVisible binding.Bool, status binding.String, state *AppState) {
	var currentPath string
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/" // Fallback for systems without a home directory
	}
	currentPath = home

	var allFiles []os.DirEntry
	visibleFiles := binding.NewUntypedList()

	pathLabel := widget.NewLabel(currentPath)
	pathLabel.Truncation = fyne.TextTruncateEllipsis

	selectedFile := ""
	selectedFileBinding := binding.NewString()
	selectedFileLabel := widget.NewLabelWithData(selectedFileBinding)
	selectedFileLabel.Wrapping = fyne.TextWrapOff      // Ensures text is on one line
	selectedFileLabel.Truncation = fyne.TextTruncateEllipsis // Truncate from the end if too long

	fileList := widget.NewListWithData(visibleFiles,
		func() fyne.CanvasObject {
			return container.NewHBox(widget.NewIcon(theme.FileIcon()), widget.NewLabel("template"))
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			item, _ := i.(binding.Untyped).Get()
			entry := item.(os.DirEntry)

			box := o.(*fyne.Container)
			icon := box.Objects[0].(*widget.Icon)
			label := box.Objects[1].(*widget.Label)

			label.SetText(entry.Name())
			if entry.IsDir() {
				icon.SetResource(theme.FolderIcon())
			} else {
				icon.SetResource(theme.FileIcon())
			}
		})

	updateFiles := func(path string) {
		currentPath = path
		pathLabel.SetText(path)

		files, err := os.ReadDir(path)
		if err != nil {
			status.Set(state.translator.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "Error.ReadingDirectory",
				TemplateData: map[string]string{
					"error": err.Error(),
				},
			}))
			return
		}
		sort.Slice(files, func(i, j int) bool {
			if files[i].IsDir() != files[j].IsDir() {
				return files[i].IsDir()
			}
			return strings.ToLower(files[i].Name()) < strings.ToLower(files[j].Name())
		})
		allFiles = files

		items := make([]interface{}, len(allFiles))
		for i, f := range allFiles {
			items[i] = f
		}
		visibleFiles.Set(items)
		fileList.Refresh()
	}

	fileList.OnSelected = func(id widget.ListItemID) {
		item, _ := visibleFiles.GetValue(id)
		entry := item.(os.DirEntry)

		newPath := filepath.Join(currentPath, entry.Name())
		if entry.IsDir() {
			updateFiles(newPath)
			selectedFile = ""
			selectedFileBinding.Set("")
		} else {
			selectedFile = newPath
			selectedFileBinding.Set(entry.Name())
		}
		fileList.UnselectAll()
	}

	updateFiles(currentPath)

	fileSearchEntry := widget.NewEntry()
	fileSearchEntry.SetPlaceHolder(state.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Search.Files"})) // Localized placeholder
	fileSearchEntry.OnChanged = func(s string) {
		s = strings.ToLower(s)
		if s == "" {
			items := make([]interface{}, len(allFiles))
			for i, f := range allFiles {
				items[i] = f
			}
			visibleFiles.Set(items)
			fileList.Refresh()
			return
		}

		filteredItems := make([]interface{}, 0)
		for _, entry := range allFiles {
			if strings.Contains(strings.ToLower(entry.Name()), s) {
				filteredItems = append(filteredItems, entry)
			}
		}
		visibleFiles.Set(filteredItems)
		fileList.Refresh()
	}

	upButton := widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() {
		parent := filepath.Dir(currentPath)
		if parent != currentPath {
			updateFiles(parent)
		}
	})

	topBar := container.NewBorder(nil, nil, upButton, nil, fileSearchEntry)
	content := container.NewBorder(
		container.NewVBox(pathLabel, topBar),
		container.NewBorder(nil, nil, widget.NewLabel(state.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Label.SelectedFile"})), nil, selectedFileLabel),
		nil, nil,
		fileList,
	)

	d := dialog.NewCustomConfirm(
		state.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Dialog.SelectFile.Title"}),
		state.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Dialog.SelectFile.Open"}),
		state.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Dialog.SelectFile.Cancel"}),
		content,
		func(confirm bool) {
			if !confirm || selectedFile == "" {
				return
			}
			go sendFile(pkg, dest, selectedFile, progress, progressVisible, status, state)
		},
		win,
	)

	d.Resize(fyne.NewSize(800, 600))
	d.Show()
}

// sendFile pushes the selected file to the specified location on the Android device.
// It updates the UI via data bindings and runs in a goroutine.
func sendFile(pkg, dest, filePath string, progress binding.Float, progressVisible binding.Bool, status binding.String, state *AppState) {
	progressVisible.Set(true)
	progress.Set(0)

	fileName := filepath.Base(filePath)
	status.Set(state.translator.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "Status.Sending",
		TemplateData: map[string]string{
			"file": fileName,
		},
	}))

	var remotePath string
	if dest == "files" {
		remotePath = fmt.Sprintf("/sdcard/Android/data/%s/files/", pkg)
	} else {
		remotePath = fmt.Sprintf("/sdcard/Android/obb/%s/", pkg)
	}

	exec.Command("adb", "shell", "mkdir", "-p", remotePath).Run()

	cmd := exec.Command("adb", "push", filePath, remotePath)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	cmd.Start()

	scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
	re := regexp.MustCompile(`\[\s*(\d+)%\]`)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			progressVal, _ := strconv.Atoi(matches[1])
			progress.Set(float64(progressVal) / 100)
		}
	}

	if err := cmd.Wait(); err != nil {
		msg := state.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Error.SendFailed"})
		status.Set(msg)
		progressVisible.Set(false)
		return
	}

	progress.Set(1)
	status.Set(state.translator.MustLocalize(&i18n.LocalizeConfig{MessageID: "Status.Success"}))

	go func() {
		time.Sleep(2 * time.Second)
		progressVisible.Set(false)
	}()
}

