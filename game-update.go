/*
//  Implementation of the Update method for the Game structure
//  This method is called once at every frame (60 frames per second)
//  by ebiten, juste before calling the Draw method (game-draw.go).
//  Provided with a few utilitary methods:
//    - CheckArrival
//    - ChooseRunners
//    - HandleLaunchRun
//    - HandleResults
//    - HandleWelcomeScreen
//    - Reset
//    - UpdateAnimation
//    - UpdateRunners
*/

package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"gitlab.univ-nantes.fr/E22B127S/projet-net-code/network"
	"log"
	"strconv"
	"strings"
	"time"
)

// HandleWelcomeScreen waits for the player to push SPACE in order to
// start the game
func (g *Game) HandleWelcomeScreen() bool {

	select { // Vérifie s'il y a une valeur dans le channel de communication du jeu
	case message := <-g.readChan: // Lorsqu'une information est reçue du serveur
		if message == network.ALL_CONNECTED { // Si le message reçu par le serveur est celui informant que tous les clients sont connectés
			g.readyNextStep = true // On indique pour le client qu'il peut passer à l'étape suivante.
		}

		if message[:1] == network.CLIENTS_IN_QUEUE && !g.readyNextStep { // Si le premier caractère du message est celui définit dans la constante définie dans network
			g.clientsCount, _ = strconv.Atoi(message[1:]) // Alors, on définit la variable de la structure game destiné à compter un nombre de joueurs à la valeur renvoyé par le serveur
		}
	default:
		// Sinon, on attend
	}

	return g.readyNextStep && inpututil.IsKeyJustPressed(ebiten.KeySpace) // La méthode renvoie true uniquement si le client à dernièrement appuyer sur espace
}

// ChooseRunners loops over all the runners to check which sprite each
// of them selected
func (g *Game) ChooseRunners() (done bool) {

	var change bool
	var direction string
	done, change, direction = g.runners[g.clientId].ManualChoose()
	if change {
		g.writeChan <- network.RUNNER_CHOICE_POSITION + fmt.Sprint(g.clientId) + direction + fmt.Sprint(g.runners[g.clientId].colorScheme)
	}
	if done && !g.responseSend { // Si le personnage du joueur a été sélectionné et qu'il n'a toujours pas envoyé l'information au serveur
		g.writeChan <- network.CLIENT_CHOOSE_RUNNER + "1" // Il envoie l'information qu'il a sélectionné son personnage au serveur
		g.responseSend = true                             // Le client enregistre qu'il a envoyé l'information au serveur
	}
	if g.responseSend && inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.responseSend = false
		g.writeChan <- network.CLIENT_CHOOSE_RUNNER + "0"
	}
	select {
	case message := <-g.readChan:
		if message == network.ALL_RUNNER_CHOOSEN { // Si le message reçu par le serveur est celui indiquant que tous les joueurs ont choisi leur personnage
			g.readyNextStep = true // On indique au client il est prêt à lancer une course
		}
		if message[:1] == network.CLIENTS_IN_QUEUE { // Si le message reçu par le serveur à comme premier caractère
			n, _ := strconv.Atoi(message[1:]) // On récupère l'information renvoyée par le serveur (entièreté du message après le premier caractère)
			g.clientsCount = 4 - n            // On affiche le nombre de joueurs restant à choisir leur joueur (joueurs totaux - joueurs ayant choisi)
		}
		if message[:1] == network.RUNNER_CHOICE_POSITION { // Si on reçoit la position d'un runner
			data := strings.Split(message[1:], " ") // On récupère les données reçues par le serveur
			clientID, _ := strconv.Atoi(data[0])    // On convertit les données de string vers des entiers
			cursorPos, _ := strconv.Atoi(data[1])
			g.runners[clientID].colorScheme = cursorPos // On affecte le runner sélectionné du clientID
		}
	default:
	}
	return g.readyNextStep
}

// HandleLaunchRun countdowns to the start of a run
func (g *Game) HandleLaunchRun() bool {

	select {
	case message := <-g.readChan: // S'il y a un message dans le channel de lecture
		if message[:1] == network.START_RACE { // Si le message indique le démarrage d'une course
			g.readyNextStep = true // On indique qu'on peut passer à l'étape suivante
		}
	default:
	}

	if g.readyNextStep { // Si on peut passer à l'étape suivante
		// On lance le décompte
		if time.Since(g.f.chrono).Milliseconds() > 1000 {
			g.launchStep++
			g.f.chrono = time.Now()
		}
		if g.launchStep >= 5 { // Si on arrive à la fin du décompte
			g.launchStep = 0       // On réinitialise le compteur
			return g.readyNextStep // On passe à l'étape suivante
		}
	}
	return false
}

// UpdateRunners loops over all the runners to update each of them
func (g *Game) UpdateRunners() {
	// Ici, on a supprimé la mise à jour des autres runners car leur position est mise à jour
	// depuis les informations reçues par le serveur et géré par le client dans une goroutine.

	g.runners[g.clientId].ManualUpdate() // Met à jour la position du runner de clientID
	if !g.runners[g.clientId].arrived {  // Si le runner ClientID n'est pas arrivé
		g.writeChan <- network.RUNNER_POSITION + fmt.Sprint(g.clientId) + fmt.Sprint(g.runners[g.clientId].xpos) + " " + fmt.Sprint(g.runners[g.clientId].speed) // Envoie la position du runner au serveur avec le clientID
	}
}

// CheckArrival loops over all the runners to check which ones are arrived
func (g *Game) CheckArrival() (gameFinish bool) {
	g.runners[g.clientId].CheckArrival(&g.f)   // On vérifie si le clientID est arrivé
	gameFinish = g.runners[g.clientId].arrived // On stocke dans gameFinish si le clientID est arrivé

	if gameFinish { // Si le joueur clientID est arrivé
		if !g.responseSend { // Si la réponse n'est pas envoyée
			g.writeChan <- network.FINISH_RACE + g.runners[g.clientId].runTime.String() // On envoie au serveur que le runner à terminer la course et on envoie son temps
			g.responseSend = true                                                       // La réponse a été envoyée
		}
	}
	return gameFinish && g.readyNextStep
}

// Reset resets all the runners and the field in order to start a new run
func (g *Game) Reset() {
	for i := range g.runners {
		g.runners[i].Reset(&g.f)
	}
	g.f.Reset()
}

// UpdateAnimation loops over all the runners to update their sprite
func (g *Game) UpdateAnimation() {
	for i := range g.runners {
		g.runners[i].UpdateAnimation(g.runnerImage)
	}
}

// HandleResults computes the results of a run and prepare them for
// being displayed
func (g *Game) HandleResults() bool {
	for i, t := range g.finalTimes {
		g.runners[i].runTime = t
	}
	if time.Since(g.f.chrono).Milliseconds() > 1000 || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.resultStep++
		g.f.chrono = time.Now()

	}
	if g.resultStep >= 4 && inpututil.IsKeyJustPressed(ebiten.KeySpace) && !g.responseSend { // Si tous les temps ont étés affichés, que la réponse n'a pas été envoyée au serveur et que le client à appuyer sur la touche espace
		g.writeChan <- network.CLIENT_WISH_RESTART // On envoie l'information au serveur que le joueur souhaite relancer une partie
		g.responseSend = true                      // La réponse a été envoyée
	}

	select {
	case message := <-g.readChan: // On vérifie qu'il y ait un message dans le channel de lecture
		if message[:1] == network.START_RACE { // Si le message indique que la partie commence (ou recommence)
			g.resultStep = 0       // On réinitialise le compteur d'affichage des temps
			g.readyNextStep = true // On indique qu'on peut passer à l'étape suivante
		}
		if message[:1] == network.CLIENTS_IN_QUEUE { // Si le message est celui qui indique combien de joueurs veulent relancer une partie
			g.clientsCount, _ = strconv.Atoi(message[1:]) // On modifie la variable qui permet de stocker cette valeur par la valeur du message
		}
	default:
	}
	return g.readyNextStep // On passe à l'étape suivante si la variable est à true
}

// Update is the main update function of the game. It is called by ebiten
// at each frame (60 times per second) just before calling Draw (game-draw.go)
// Depending on the current state of the game it calls the above utilitary
// function, and then it may update the state of the game
func (g *Game) Update() error {
	// Différentes variables utilisées à plusieurs endroits du programme sont réinitialisées
	// à chaque fois qu'on passe à une étape suivante
	switch g.state {
	case StateWelcomeScreen:
		done := g.HandleWelcomeScreen()
		if done {
			g.state++
			g.readyNextStep = false
			g.responseSend = false
			g.clientsCount = 0
			// On demande au serveur de nous envoyer un compteur pour l'afficher en arrivant sur la page
			// Permet d'avoir directement le nombre de personnes n'ayant pas choisi leur personnage sans attendre
			// qu'un joueur en choisisse un pour actualiser ce compteur coté client.
			g.writeChan <- network.NEED_CLIENT_COUNT
		}
	case StateChooseRunner:
		done := g.ChooseRunners()
		if done {
			g.UpdateAnimation()
			g.state++
			g.readyNextStep = false
			g.responseSend = false
			g.clientsCount = 0
		}
	case StateLaunchRun:
		done := g.HandleLaunchRun()
		if done {
			g.state++
			g.readyNextStep = false
			g.responseSend = false
			g.clientsCount = 0
			go g.multiplayerGame() // Lancement de la goroutine permettant de jouer en multijoueurs
		}
	case StateRun:
		g.UpdateRunners()
		finished := g.CheckArrival()
		g.UpdateAnimation()
		if finished {
			g.state++
			g.readyNextStep = false
			g.responseSend = false
			g.clientsCount = 0
		}
	case StateResult:
		done := g.HandleResults()
		if done {
			g.Reset()
			g.state = StateLaunchRun
			g.readyNextStep = false
			g.responseSend = false
			g.clientsCount = 0
		}
	}
	return nil
}

// multiplayerGame permet de mettre à jour la position des joueurs autres que clientID
// depuis une goroutine et récupère les temps des autres joueurs à la fin d'une course.
func (g *Game) multiplayerGame() {
	var message string
	for !g.readyNextStep { // Tant qu'on ne peut pas passer à l'étape suivante
		message = <-g.readChan // On lit le message du channel
		switch {
		case message[:1] == network.RUNNER_POSITION: // Si le message reçu est celui qui indique la position d'un joueur
			var playerID, _ = strconv.Atoi(message[1:2]) // On récupère l'identifiant du joueur en second caractère de la chaine
			var data = strings.Split(message[2:], " ")
			var playerPos, _ = strconv.ParseFloat(data[0], 64) // On convertit la position recue en float
			var playerSpeed, _ = strconv.ParseFloat(data[1], 64)
			var runner = &g.runners[playerID] // On stocke temporairement le runner qui va être mis à jour

			if !runner.arrived && playerID != g.clientId { // Si le runner n'est pas arrivé et que son identifiant est différent de celui du clientID
				runner.xpos = playerPos // Sa position est mise à jour avec la position recue par le serveur
				runner.speed = playerSpeed
				runner.CheckArrival(&g.f) // On check si le runner est arrivé
			}
		case message[:1] == network.FINISH_RACE: // Si le message reçu est celui qui indique que la course est terminée
			var times = strings.Split(message[1:], " ") // On split la chaine de caractère recue pour en faire un tableau de temps (au format string)
			for i, t := range times {                   // On parcourt les temps
				log.Println(i, t)
				g.finalTimes[i], _ = time.ParseDuration(t) // On parse le temps t du tableau en string et on définit le temps du runner i par celui parsé
				g.runners[i].arrived = true                // Par sécurité, on force l'arrivée du runner
			}
			g.readyNextStep = true // On indique qu'on peut passer à la suite
		default:
		}
	}
	// On doit terminer cette goroutine lorsqu'on a reçu les temps du serveur, car sinon
	// les autres messages provenant du serveur peuvent être récupéré dans cette goroutine et
	// ne pourront pas être traités, la partie ne pourra pas continuer et le joueur sera bloqué.
}
