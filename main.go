package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

var (
	inputArg      = flag.String("input", "", "The game of life file to parse")
	iterationsArg = flag.Int("iterations", 0, "The number of iterations to run")
)

const (
	FILE_HEADER = "#Life 1.06"
)

type Cell struct {
	x, y int64
}

func (cell Cell) neighbors() <-chan Cell {
	neighborsCh := make(chan Cell)

	yieldForX := func(x int64) {
		if cell.y > math.MinInt64 {
			neighborsCh <- Cell{x, cell.y - 1}
		}
		if cell.x != x {
			neighborsCh <- Cell{x, cell.y}
		}
		if cell.y < math.MaxInt64 {
			neighborsCh <- Cell{x, cell.y + 1}
		}
	}

	go func() {
		if cell.x > math.MinInt64 {
			yieldForX(cell.x - 1)
		}
		yieldForX(cell.x)

		if cell.x < math.MaxInt64 {
			yieldForX(cell.x + 1)
		}

		close(neighborsCh)
	}()

	return neighborsCh
}

type Cells map[Cell]struct{}

func (cells Cells) numAliveNeighbors(cell Cell) uint8 {
	aliveCount := uint8(0)
	for neighbor := range cell.neighbors() {
		if cells.hasCell(neighbor) {
			aliveCount++
		}
	}
	return aliveCount
}

func (cells Cells) deadNeighbors() <-chan Cell {
	deadNeighborsCh := make(chan Cell)
	go func() {
		deadNeighborCells := make(Cells)
		for cell := range cells {
			for neighbor := range cell.neighbors() {
				if !cells.hasCell(neighbor) { // neighbor is dead
					deadNeighborCells.addCell(neighbor)
				}
			}
		}

		for deadNeighbor := range deadNeighborCells {
			deadNeighborsCh <- deadNeighbor
		}

		close(deadNeighborsCh)
	}()
	return deadNeighborsCh
}

func (cells Cells) addCell(cell Cell) {
	cells[cell] = struct{}{}
}

func (cells Cells) hasCell(cell Cell) bool {
	_, found := cells[cell]
	return found
}

func (cells Cells) removeCell(cell Cell) {
	delete(cells, cell)
}

func parseCells(inputFile string) (Cells, error) {
	file, err := os.Open(inputFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cells := make(Cells)

	headerFound := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			if line == FILE_HEADER && len(cells) == 0 {
				headerFound = true
			}
			continue
		}

		cell := Cell{}
		items, err := fmt.Fscanf(strings.NewReader(line), "%d %d", &cell.x, &cell.y)
		if items < 2 || err != nil {
			return nil, fmt.Errorf("failed to parse line '%d', %v", len(cells)+1, err)
		}
		cells.addCell(cell)
	}

	if !headerFound {
		return nil, fmt.Errorf("Invalid Game of Life file: needed %s indicator as first line", FILE_HEADER)
	}

	return cells, nil
}

func printCells(w io.Writer, cells Cells) error {
	if _, err := fmt.Fprintf(w, "%s\n\n", FILE_HEADER); err != nil {
		return err
	}
	for cell := range cells {
		if _, err := fmt.Fprintf(w, "%d %d\n", cell.x, cell.y); err != nil {
			return err
		}
	}
	return nil
}

func runGameOfLife(inputFile string, iterations int) error {
	cells, err := parseCells(*inputArg)
	if err != nil {
		return fmt.Errorf("parsing cells failed: %v", err)
	}

	// Run simultion
	for iteration := 0; iteration < iterations; iteration++ {
		// If an "alive" cell had less than 2 or more than 3 alive neighbors (in any of the 8 surrounding cells), it becomes dead.
		dyingCells := make(Cells)
		for cell := range cells {
			aliveNeighbors := cells.numAliveNeighbors(cell)
			if aliveNeighbors < 2 || aliveNeighbors > 3 {
				dyingCells.addCell(cell)
			}
		}

		// If a "dead" cell had *exactly* 3 alive neighbors, it becomes alive.
		birthedCells := make(Cells)
		for cell := range cells.deadNeighbors() {
			aliveNeighbors := cells.numAliveNeighbors(cell)
			if aliveNeighbors == 3 {
				birthedCells.addCell(cell)
			}
		}

		// apply changes for next iteration
		fmt.Fprintf(os.Stderr, "Iteration #%d\n", iteration)
		for cell := range dyingCells {
			fmt.Fprintf(os.Stderr, "(%d, %d) is dying\n", cell.x, cell.y)
			cells.removeCell(cell)
		}
		for cell := range birthedCells {
			fmt.Fprintf(os.Stderr, "(%d, %d) is being born\n", cell.x, cell.y)
			cells.addCell(cell)
		}
	}

	if err := printCells(os.Stdout, cells); err != nil {
		return fmt.Errorf("printing cells failed: %v", err)
	}

	return nil
}

func main() {
	flag.Parse()

	if err := runGameOfLife(*inputArg, *iterationsArg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run Game of Life, err='%v'", err)
		os.Exit(1)
	}
}
