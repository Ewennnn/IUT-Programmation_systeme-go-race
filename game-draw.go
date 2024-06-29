/*
//  Implementation of the Draw method for the Game structure
//  This method is called once at every frame (60 frames per second)
//  by ebiten, juste after calling the Update method (game-update.go)
//  Provided with a few utilitary methods:
//    - DrawLaunch
//    - DrawResult
//    - DrawRun
//    - DrawSelectScreen
//    - DrawWelcomeScreen
*/

package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// DrawWelcomeScreen displays the title screen in the game window
func (g *Game) DrawWelcomeScreen(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(
		screen,
		fmt.Sprint("Track & Field: BUT2 2022-2023 Edition"),
		screenWidth/2-120,
		screenHeight/2-20,
	)
	// Cette méthode est appelée 60 fois par secondes
	// Affiche le nombre de clients connectés sur la page d'acceuil.
	if g.clientsCount >= 4 {
		ebitenutil.DebugPrintAt(
			screen,
			fmt.Sprint("Press SPACE to play"),
			screenWidth/2-60,
			screenHeight/2+10,
		)
		ebitenutil.DebugPrintAt(
			screen,
			"All players are connected !",
			screenWidth/2-80,
			screenHeight/2+40,
		)
	} else {
		ebitenutil.DebugPrintAt(
			screen,
			"Waiting players "+fmt.Sprint(g.clientsCount),
			screenWidth/2-60,
			screenHeight/2+10,
		)
	}
}

// DrawSelectScreen displays the runner selection screen in the game window
func (g *Game) DrawSelectScreen(screen *ebiten.Image) {

	ebitenutil.DebugPrintAt(
		screen,
		fmt.Sprint("Select your player"),
		screenWidth/2-60,
		20,
	)
	ebitenutil.DebugPrintAt(
		screen,
		fmt.Sprint("remaining players "+fmt.Sprint(g.clientsCount)),
		screenWidth/2-60,
		screenHeight-20,
	)

	xStep := (screenWidth - 100) / 8
	xPadding := (xStep - 32) / 2
	yPos := (screenHeight - 32) / 2
	for i := 0; i < 8; i++ {
		options := &ebiten.DrawImageOptions{}
		xPos := 50 + i*xStep + xPadding
		options.GeoM.Translate(float64(xPos), float64(yPos))
		screen.DrawImage(g.runnerImage.SubImage(image.Rect(0, i*32, 32, i*32+32)).(*ebiten.Image), options)
	}
	for i := range g.runners {
		g.runners[i].DrawSelection(screen, xStep, i)
	}
}

// DrawLaunch displays the countdown before a run in the game window
func (g *Game) DrawLaunch(screen *ebiten.Image) {
	if g.launchStep > 1 {
		ebitenutil.DebugPrintAt(screen, fmt.Sprint(5-g.launchStep), screenWidth/2-10, 10)
	}
}

// DrawRun displays the current state of the run in the game window
func (g *Game) DrawRun(screen *ebiten.Image, drawChrono bool) {
	g.f.Draw(screen, drawChrono)
	for i := range g.runners {
		g.runners[i].Draw(screen)
	}
}

// DrawResult displays the results of the run in the game window
func (g *Game) DrawResult(screen *ebiten.Image) {
	ranking := [4]int{-1, -1, -1, -1}
	for i := range g.runners {
		rank := 0
		for j := range g.runners {
			if g.runners[i].runTime > g.runners[j].runTime {
				rank++
			}
		}
		for ranking[rank] != -1 {
			rank++
		}
		ranking[rank] = i
	}

	for i := 1; i < g.resultStep && i <= 4; i++ {
		s, ms := GetSeconds(g.runners[ranking[i-1]].runTime.Milliseconds())
		ebitenutil.DebugPrintAt(screen, fmt.Sprint(i, ". P", ranking[i-1], "     ", s, ":", ms), screenWidth/2-40, 55+ranking[i-1]*20)
	}

	if g.resultStep > 4 {
		if g.clientsCount == 4 {
			ebitenutil.DebugPrintAt(
				screen,
				fmt.Sprint("Game will be restarted soon..."),
				screenWidth/2-60,
				screenHeight/2+10,
			)
		} else {
			var text string
			if g.responseSend {
				text = "    You are ready !\n   Waiting players " + fmt.Sprint(4-g.clientsCount)
			} else {
				text = "Press SPACE to restart !\n   Waiting players " + fmt.Sprint(4-g.clientsCount)
			}
			ebitenutil.DebugPrintAt(screen, text, screenWidth/2-60, 10)
		}
	}
}

// Draw is the main drawing function of the game. It is called by ebiten at
// each frame (60 times per second) just after calling Update (game-update.go)
// Depending on the current state of the game it calls the above utilitary
// function to draw what is needed in the game window
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{141, 200, 235, 255})

	if g.getTPS {
		ebitenutil.DebugPrint(screen, fmt.Sprint(ebiten.CurrentTPS()))
	}

	switch g.state {
	case StateWelcomeScreen:
		g.DrawWelcomeScreen(screen)
	case StateChooseRunner:
		g.DrawSelectScreen(screen)
	case StateLaunchRun:
		g.DrawLaunch(screen)
		g.DrawRun(screen, false)
	case StateRun:
		g.DrawRun(screen, true)
	case StateResult:
		g.DrawResult(screen)
		g.DrawRun(screen, false)
	}
}
