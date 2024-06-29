/*
// Implementation of a main function setting a few characteristics of
// the game window, creating a game, and launching it
*/

package main

import (
	"flag"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"image"
	_ "image/png"
	"log"
)

const (
	screenWidth  = 800 // Width of the game window (in pixels)
	screenHeight = 160 // Height of the game window (in pixels)
)

func main() {

	var getTPS bool
	flag.BoolVar(&getTPS, "tps", false, "Afficher le nombre d'appel à Update par seconde")
	flag.Parse()

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Waiting connection...")
	// Ajout d'une icon à la fenêtre
	var eimage, _, _ = ebitenutil.NewImageFromFile("./assets/golang.png")
	ebiten.SetWindowIcon([]image.Image{eimage})

	g := InitGame("localhost", "8080")
	g.getTPS = getTPS

	err := ebiten.RunGame(&g)
	log.Print(err)
}
