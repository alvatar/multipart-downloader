package main

import (
	"fmt"
	md "github.com/alvatar/multipart-downloader"
	"github.com/sethgrid/multibar"
)

// Progress type
type progress struct {
	progressBars   *multibar.BarContainer
}

// Setup progress visualization
func NewProgress(chunks []md.Chunk) (prog *progress) {
	numChunks := len(chunks)
	pBars, _ := multibar.New()

	prog = &progress{
		progressBars: pBars,
	}

	for i := 0; i < numChunks; i++ {
		prog.progressBars.MakeBar(int(chunks[i].End - chunks[i].Begin), fmt.Sprintf("%2d:", i+1))
	}

	go prog.progressBars.Listen()

	return
}

// Update values from connections progress
func (prog *progress) Update(progressArray []md.ConnectionProgress) {
	for i := 0; i < len(progressArray); i++ {
		relativeProgress := int(progressArray[i].Current - progressArray[i].Begin)
		prog.progressBars.Bars[i].Update(relativeProgress)
	}
}

