package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func buildWorld(world [][]byte, index, restOfRows, workerHeight, imageWidth int, numOfWorker int, sendByte chan byte){
	//make a new world
	switch index{
	case 0:
		workPlace := make([][]byte,workerHeight + 2)
		for i := range workPlace{
			workPlace[i] = make([]byte, imageWidth)
		}

		workPlace[0] = world[len(world) - 1]
		for y := 1; y < workerHeight + 2; y++{
			for x := 0; x < imageWidth; x++{
				workPlace[y][x] = world[index * workerHeight + y - 1][x]
			}
		}

		for y := 0; y < workerHeight + 2; y ++{
			for x := 0; x < imageWidth; x++{
				sendByte <- workPlace[y][x]
			}
		}

	case numOfWorker - 1:

		if restOfRows != 0{
			workerHeight += restOfRows
		}

		workPlace := make([][]byte,workerHeight + 2)
		for i := range workPlace{
			workPlace[i] = make([]byte, imageWidth)
		}

		workPlace[workerHeight + 1] = world[0]

		for y := 0; y < workerHeight + 1; y++{
			for x := 0; x < imageWidth; x++{
				workPlace[y][x] = world[index * (workerHeight - restOfRows) + y - 1][x]
			}
		}

		for y := 0; y < workerHeight + 2; y ++{
			for x := 0; x < imageWidth; x++{
				sendByte <- workPlace[y][x]
			}
		}

	default:
		workPlace := make([][]byte,workerHeight + 2)
		for i := range workPlace{
			workPlace[i] = make([]byte, imageWidth)
		}

		for y := 0; y < workerHeight + 2; y++{
			for x := 0; x < imageWidth; x++{
				workPlace[y][x] = world[index * workerHeight + y - 1][x]
			}
		}

		for y := 0; y < workerHeight + 2; y ++{
			for x := 0; x < imageWidth; x++{
				sendByte <- workPlace[y][x]
			}
		}
	}
}
//a function that deal with every single world parts and return the next turn of this part
//the output with height + 2
func worker(workerHeight, imageWidth int, sendByte chan byte, out chan byte){

	workPlace := make([][]byte,workerHeight + 2)
	for i := range workPlace{
		workPlace[i] = make([]byte, imageWidth)
	}


	for y := 0; y < workerHeight + 2; y++{
		for x := 0; x < imageWidth; x++{
			currentByte := <- sendByte
			workPlace[y][x] = currentByte
		}
	}
	nextWorldPart := schrodinger(workPlace)

	for y := 1; y < workerHeight + 1; y++{
		for x := 0; x < imageWidth; x++{
			out <- nextWorldPart[y][x]
		}
	}
}

//the logic of the Game Of Live
//i call it schrodinger because if you dont observe the cat(which is the world containing cells),
// u will never know whether it is alive
func schrodinger(cat [][]byte)[][]byte{
	imageHeight := len(cat)
	imageWidth := len(cat[0])
	nextWorld := make([][]byte, imageHeight)
	for i := range nextWorld {
		nextWorld[i] = make([]byte, imageWidth)
	}
	//create a for loop go through all cells in the world
	for y := 1; y < imageHeight - 1; y++ {
		for x := 0; x < imageWidth; x++ {
			//create a int value that counts how many alive neighbours does a cell have
			aliveNeighbours := 0
			//extract the 3x3 matrix which centred at the cell itself
			//go through every neighbour and count the aliveNeighbours
			for i := -1; i < 2; i++{
				for j := -1; j < 2; j++{
					if i == 0 && j == 0{continue}                                              //I don't care if the cell itself is alive or dead at this stage
					if cat[y + i][(x + j + imageWidth) % imageWidth] == 255{                  //if there is an alive neighbour, the count of alive neighbours increase by 1
						aliveNeighbours += 1
					}
				}
			}
			if cat[y][x] == 255{
				if aliveNeighbours < 2 || aliveNeighbours > 3{                  //if the cell itself is alive, check the neighbours:
					nextWorld[y][x] = 0                                         //if it has <2 or>3 alive neighbours, it will die in nextWorld :(
				} else{nextWorld[y][x] = 255}                                   //if it has =2 or =3 alive neighbours, it will survive in nextWorld :)
			}
			if cat[y][x] == 0{
				if aliveNeighbours == 3{                                        //if the cell itself is dead, check the neighbours:
					nextWorld[y][x] = 255                                       //if it has =3 neighbours, it will become alive in nextWorld ;)
				}else{nextWorld[y][x] = 0}
			}
		}
	}
	return nextWorld
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell, key chan rune) {

	// Create the 2D slice to store the world.
	world := make([][]byte, p.imageHeight)
	for i := range world {
		world[i] = make([]byte, p.imageWidth)
	}

	// Request the io goroutine to read in the image with the given filename.
	d.io.command <- ioInput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")

	// The io goroutine sends the requested image byte by byte, in rows.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			val := <-d.io.inputVal
			if val != 0 {
				fmt.Println("Alive cell at", x, y)
				world[y][x] = val
			}
		}
	}

	// Calculate the new state of Game of Life after the given number of turns.
	out := make([]chan byte, p.threads)
	sendByte := make(chan byte)
	ticker := time.NewTicker(2 * time.Second)

	restOfRows := p.imageHeight % p.threads
	workerHeight := p.imageHeight/p.threads

	running := true
	for turns := 0; turns < p.turns && running ; turns++ {

	select{

	case pressedKey := <- key:
		if pressedKey == 's'{
			d.io.command <- ioOutput
			d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight), strconv.Itoa(turns)}, "x")
			d.io.outputWorld <- world
		}else if pressedKey == 'q'{
			d.io.command <- ioOutput
			d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight), strconv.Itoa(turns)}, "x")
			d.io.outputWorld <- world
			fmt.Println("program terminating...")
			running = false
		}else if pressedKey == 'p'{
			fmt.Println(turns)
			for{
				pausing := <- key
				if pausing == 'p'{
					fmt.Println("continuing...")
					break
				}else{
					continue}
			}
		}

	case <- ticker.C:
		var finalAlive []cell
		// Go through the world and append the cells that are still alive.
		for y := 0; y < p.imageHeight; y++ {
			for x := 0; x < p.imageWidth; x++ {
				if world[y][x] != 0 {
					finalAlive = append(finalAlive, cell{x: x, y: y})
				}
			}
		}
		fmt.Println("number of alive cells is:", len(finalAlive))

	default:

	}

		for i := 0; i < p.threads; i++{
			out[i] = make(chan byte)

			if i == p.threads - 1{
				go worker(workerHeight + restOfRows, p.imageWidth, sendByte, out[i])
			}else{
				go worker(workerHeight, p.imageWidth, sendByte, out[i])
			}
			buildWorld(world, i, restOfRows, workerHeight, p.imageWidth, p.threads, sendByte)
		}


		for i := 0; i < p.threads; i++{
			if i != p.threads - 1{
				newWorldPart := make([][]byte, workerHeight)
				for i := range newWorldPart{
					newWorldPart[i] = make([]byte, p.imageHeight)
				}

				for y := 0 ; y < workerHeight; y++{
					for x := 0; x < p.imageWidth; x++{
						newWorldPart[y][x] = <- out[i]
					}
				}

				for y := 0; y < workerHeight; y++{
					for x := 0; x < p.imageWidth; x++{
						world[workerHeight * i + y][x] = newWorldPart[y][x]
					}
				}
			}else{
				newWorldPart := make([][]byte, workerHeight + restOfRows)
				for i := range newWorldPart{
					newWorldPart[i] = make([]byte, p.imageWidth)
				}

				for y := 0 ; y < workerHeight + restOfRows; y++{
					for x := 0; x < p.imageWidth; x++{
						newWorldPart[y][x] = <- out[i]
					}
				}

				for y := 0; y < workerHeight + restOfRows; y++{
					for x := 0; x < p.imageHeight; x++{
						world[workerHeight * i + y][x] = newWorldPart[y][x]
					}
				}
			}
		}
	}



	// Create an empty slice to store coordinates of cells that are still alive after p.turns are done.
	var finalAlive []cell
	// Go through the world and append the cells that are still alive.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			if world[y][x] != 0 {
				finalAlive = append(finalAlive, cell{x: x, y: y})
				fmt.Println(x, y)
			}
		}
	}

	d.io.command <- ioOutput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")
	d.io.outputWorld <- world
	// Make sure that the Io has finished any output before exiting.
	d.io.command <- ioCheckIdle
	<-d.io.idle

	// Return the coordinates of cells that are still alive.
	alive <- finalAlive
}