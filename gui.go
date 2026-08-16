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

const (
	screenWidth  = 400
	screenHeight = 150
)

type ChunkerGUI struct {
	mergeButton        *widget.Clickable
	splitButton        *widget.Clickable
	chunkSizeRadio     *widget.Enum
	chunkSizeKB        int
	activeMessageLabel string
	chunkSizeEditor    *widget.Editor

	exp *explorer.Explorer
}

func GUIRun() {

	go func() {
		w := new(app.Window)
		w.Option(app.Size(unit.Dp(screenWidth), unit.Dp(screenHeight)))
		chnk := &ChunkerGUI{}
		chnk.setDefaults(w)
		if err := chnk.GUILoop(w); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}

func (c *ChunkerGUI) setDefaults(w *app.Window) {
	c.mergeButton = new(widget.Clickable)
	c.splitButton = new(widget.Clickable)
	c.chunkSizeRadio = new(widget.Enum)
	c.chunkSizeEditor = new(widget.Editor)

	c.chunkSizeKB = 0

	c.activeMessageLabel = ""

	c.chunkSizeRadio.Value = "M"
	c.chunkSizeEditor.SetText("10")
	c.chunkSizeEditor = &widget.Editor{
		SingleLine: true,
		Submit:     true,
	}

	c.exp = explorer.NewExplorer(w)
}

func (c *ChunkerGUI) GUILoop(w *app.Window) error {
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
				_, ok := c.chunkSizeEditor.Update(gtx)
				if !ok {
					break
				}
				hasTextUpdate = true
			}

			chunkSizeText := c.chunkSizeEditor.Text()
			if radioButtonsGroup.Update(gtx) || hasTextUpdate {
				parsed, err := parseChunkSize(fmt.Sprintf("%s%s", chunkSizeText, c.chunkSizeRadio.Value))
				if err != nil {
					if chunkSizeText == "" {
						c.activeMessageLabel = "Must input a chunk size" //TODO make these strings const
					} else {
						c.activeMessageLabel = err.Error()
						fmt.Println(err)
					}
					validChunkSize = false
				} else {
					c.chunkSizeKB = parsed
					validChunkSize = true
				}
			}

			c.processMergeButton(gtx)
			c.processSplitButton(gtx, validChunkSize)

			c.GUIRoot(gtx, th)
			e.Frame(gtx.Ops)
		}
	}
}

func (c *ChunkerGUI) processSplitButton(gtx layout.Context, validChunkSize bool) {
	if c.splitButton.Clicked(gtx) {

		if validChunkSize {
			go func() {
				file, err := c.exp.ChooseFile()
				if err != nil {
					fmt.Printf("ChooseFile err: %s", err.Error())
					return
				}

				defer file.Close()

				if validChunkSize {
					err = splitFile(file, c.chunkSizeKB)
					if err != nil {
						fmt.Println(err)
						c.activeMessageLabel = err.Error()
						return
					}

					c.activeMessageLabel = "Split success"
				}
			}()
		}
	}
}

func (c *ChunkerGUI) processMergeButton(gtx layout.Context) {
	if c.mergeButton.Clicked(gtx) {

		go func() {
			files, err := c.exp.ChooseFiles("")
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
				c.activeMessageLabel = err.Error()
				return
			}

			c.activeMessageLabel = "Merge success"

		}()
	}
}

func (c *ChunkerGUI) GUIRoot(gtx layout.Context, th *material.Theme) layout.Dimensions {

	widgets := []layout.Widget{
		func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle, Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return material.Label(th, unit.Sp(12), c.activeMessageLabel).Layout(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							e := material.Editor(th, c.chunkSizeEditor, "Chunk size")
							e.Font.Style = font.Regular
							border := widget.Border{Color: color.NRGBA{A: 0xff}, CornerRadius: unit.Dp(8), Width: unit.Dp(2)}
							return border.Layout(gtx, func(gtx C) D {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, e.Layout)
							})
						}),
						layout.Rigid(func(gtx C) D {
							return layout.Flex{}.Layout(gtx,
								layout.Rigid(material.RadioButton(th, c.chunkSizeRadio, "K", "KB").Layout),
								layout.Rigid(material.RadioButton(th, c.chunkSizeRadio, "M", "MB").Layout),
								layout.Rigid(material.RadioButton(th, c.chunkSizeRadio, "G", "GB").Layout),
							)
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx, //also try space evenly once we get this grouped with the above
							layout.Rigid(func(gtx C) D {
								btn := material.Button(th, c.splitButton, "Split")
								return btn.Layout(gtx)
							}),
							layout.Rigid(func(gtx C) D {
								return layout.Spacer{Width: unit.Dp(16)}.Layout(gtx)
							}),
							layout.Rigid(func(gtx C) D {
								btn := material.Button(th, c.mergeButton, "Merge")
								return btn.Layout(gtx)
							}),
						)
					})
				}),
			)
		},
	}

	return layout.Stack{Alignment: layout.Center}.Layout(gtx,
		layout.Expanded(
			func(gtx C) D {
				return material.List(th, list).Layout(gtx, len(widgets), func(gtx C, i int) D {
					return layout.UniformInset(unit.Dp(16)).Layout(gtx, widgets[i])
				})
			}),
	)

}
