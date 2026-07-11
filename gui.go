package main

import (
	"fmt"
	"image/color"
	"os"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"
)

//TODO put all this in a struct

var (
	mergeButton = new(widget.Clickable)
	splitButton = new(widget.Clickable)

	chunkSizeRadio = new(widget.Enum)

	chunkSizeKB = 0

	messageLabel    = ""
	chunkSizeEditor = &widget.Editor{
		SingleLine: true,
		Submit:     true,
	}

	exp *explorer.Explorer
)

func GUIRun() {

	go func() {
		w := new(app.Window)
		exp = explorer.NewExplorer(w)
		w.Option(app.Size(unit.Dp(800), unit.Dp(700)))
		setDefaults()
		if err := GUILoop(w); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}

func setDefaults() {
	chunkSizeRadio.Value = "M"
	chunkSizeEditor.SetText("10")
}

func GUILoop(w *app.Window) error {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	//TODO add a progress bar that tracks how far through the program we are

	validChunkSize := false

	var ops op.Ops
	for {
		e := w.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			if *disable {
				gtx = gtx.Disabled()
			}

			hasTextUpdate := false
			for {
				_, ok := chunkSizeEditor.Update(gtx)
				if !ok {
					break
				}
				hasTextUpdate = true
			}

			if radioButtonsGroup.Update(gtx) || hasTextUpdate {
				parsed, err := parseChunkSize(fmt.Sprintf("%s%s", chunkSizeEditor.Text(), chunkSizeRadio.Value))
				if err != nil {
					messageLabel = err.Error()
					fmt.Println(err)
					validChunkSize = false
				} else {
					chunkSizeKB = parsed
					validChunkSize = true
				}
			}

			processMergeButton(gtx)
			processSplitButton(gtx, validChunkSize)

			GUIRoot(gtx, th)
			e.Frame(gtx.Ops)
		}
	}
}

func processSplitButton(gtx layout.Context, validChunkSize bool) {
	if splitButton.Clicked(gtx) {

		if validChunkSize {
			go func() {
				file, err := exp.ChooseFile()
				if err != nil {
					fmt.Printf("ChooseFile err: %s", err.Error())
					return
				}

				defer file.Close()

				if validChunkSize {
					err = splitFile(file, chunkSizeKB)
					if err != nil {
						fmt.Println(err)
						messageLabel = err.Error()
						return
					}

					messageLabel = "Split success"
				}
			}()
		}
	}
}

func processMergeButton(gtx layout.Context) {
	if mergeButton.Clicked(gtx) {

		go func() {
			files, err := exp.ChooseFiles("")
			if err != nil {
				fmt.Printf("ChooseFiles err: %s", err.Error())
				return
			}

			defer func() {
				for _, file := range files {
					file.Close()
				}
			}()

			err = assembleFile(files)
			if err != nil {
				fmt.Println(err)
				messageLabel = err.Error()
				return
			}

			messageLabel = "Merge success"

		}()
	}
}

func GUIRoot(gtx layout.Context, th *material.Theme) layout.Dimensions {

	/*
		Layout TODO:
			-change the layout in concordence with reef's ideas
			-decrease the viewport size
	*/

	widgets := []layout.Widget{
		func(gtx C) D {
			return material.H3(th, messageLabel).Layout(gtx)
		}, //TODO replace the label with a dialog box
		func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					e := material.Editor(th, chunkSizeEditor, "Chunk size")
					e.Font.Style = font.Regular
					border := widget.Border{Color: color.NRGBA{A: 0xff}, CornerRadius: unit.Dp(8), Width: unit.Dp(2)}
					return border.Layout(gtx, func(gtx C) D {
						return layout.UniformInset(unit.Dp(8)).Layout(gtx, e.Layout)
					})
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(material.RadioButton(th, chunkSizeRadio, "K", "KB").Layout),
						layout.Rigid(material.RadioButton(th, chunkSizeRadio, "M", "MB").Layout),
						layout.Rigid(material.RadioButton(th, chunkSizeRadio, "G", "GB").Layout),
					)
				}),
			)
		},
		func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx, //also try space evenly once we get this grouped with the above
				layout.Rigid(func(gtx C) D {
					btn := material.Button(th, splitButton, "Split")
					return btn.Layout(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					btn := material.Button(th, mergeButton, "Merge")
					return btn.Layout(gtx)
				}),
			)
		},
	}

	//TODO make widgets stack in the center
	return layout.Stack{Alignment: layout.Center}.Layout(gtx,
		layout.Expanded(
			func(gtx C) D {
				return material.List(th, list).Layout(gtx, len(widgets), func(gtx C, i int) D {
					return layout.UniformInset(unit.Dp(16)).Layout(gtx, widgets[i])
				})
			}),
	)

}
