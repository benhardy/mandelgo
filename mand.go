package main

import "github.com/fogleman/gg"
import "math"
import "fmt"
import "runtime"
import "flag"

type color struct {
    r, g, b float64
}

func (col *color) smooth(other color, howFar float64) color {
    return color {
        col.r + howFar * (other.r - col.r),
        col.g + howFar * (other.g - col.g),
        col.b + howFar * (other.b - col.b),
    }
}

type pixel struct {
    x int
    y int
    paint color
}

func main() {
    palette := []color{
        color { 0.00, 0.00, 0.00 }, // black
        color { 0.80, 0.05, 0.05 }, // red
        color { 0.20, 0.50, 0.10 }, // lime
        color { 0.40, 0.40, 0.45 }, // slate
        color { 0.80, 0.90, 0.10 }, // gold
        color { 0.17, 0.04, 0.30 }, // violet
        color { 0.02, 0.25, 0.01 }, // grass
        color { 0.95, 0.97, 0.96 }, // cream
        color { 0.70, 1.00, 0.00 }, // chartreuse
        color { 0.22, 0.63, 0.30 }, // dark chartreuse
        color { 0.50, 0.00, 0.13 }, // burgundy
        color { 0.968, 0.913, 0.556 }, // flavescent yellow
        color { 0.50, 0.70, 1.00 }, // sky
        color { 0.074, 0.533, 0.031 }, // india green
        color { 0.513, 0.411, 0.325 }, // pastel brown
        color { 0.854, 0.196, 0.529 }, // deep cerise pink
        color { 0.40, 0.00, 0.00 }, // deep red
        color { 1.00, 1.00, 1.00 }, // white
    }
    widthPtr   := flag.Int("width", 2000, "image width in pixels")
    heightPtr  := flag.Int("height", 1200, "image height in pixels")
    iMaxPtr    := flag.Int("iterations", 3000, "maximum number of iterations")
    xCenterPtr := flag.Float64("real", -0.745, "real component of image center")
    yCenterPtr := flag.Float64("imag", 0.149,  "imaginary component of image center")
    scalePtr   := flag.Float64("scale", 0.03, "width of view window in complex plane")
    filePtr    := flag.String("save", "out.png", "name of file to save image to")
    flag.Parse()
    width, height := *widthPtr, *heightPtr
    iMax := *iMaxPtr
    xCenter, yCenter := *xCenterPtr, *yCenterPtr
    scale := *scalePtr
    dc      := gg.NewContext(width, height)
    aspect  := float64(height) / float64(width)
    parallelism := runtime.NumCPU()
    done := make(chan int)
    painter := make(chan pixel)
    fmt.Println("firing up threads")
    for threadNum := 0; threadNum < parallelism; threadNum++ {
        go func(thread int) {
            for y := thread; y < height; y+=parallelism {
                for x := 0; x < width; x++ {
                    xc := xCenter + scale * (float64(x) / float64(width) - 0.5)
                    yc := yCenter - aspect * scale * (float64(y) / float64(height) - 0.5)
                    xp := 0.0
                    yp := 0.0
                    var i int
                    escaped := false
                    xp2 := 0.0
                    yp2 := 0.0
                    for i = 0; i < iMax; i++ {
                        xp2 = xp * xp
                        yp2 = yp * yp
                        if xp2 + yp2 > 4.0 {
                            escaped = true
                            break
                        }
                        xp_ := xp2 - yp2 + xc
                        yp = 2 * xp * yp + yc
                        xp = xp_
                        //xp, yp = xp2 - yp2 + xc, 2 * xp * yp + yc
                    }
                    var paint color
                    if escaped {
                        logZn := math.Log(xp2 + yp2) / 2.0
                        nu := math.Log(logZn / math.Log(2) ) / math.Log(2)
                        iSmooth := float64(i) - nu + 1
                        palPos := float64(len(palette)-1) * math.Log(iSmooth) / math.Log(float64(iMax))
                        palBase := int(palPos)
                        palIndex := math.Floor(palPos)
                        paint = palette[palBase].smooth(palette[palBase + 1], palPos - palIndex)
                    } else {
                        paint = color { 0.0, 0.0, 0.0 }
                    }
                    pix := pixel { x, y, paint }
                    painter <- pix
                }
            }
            done <- thread
        }(threadNum)
    }
    totalPixels := width * height
    // painter
    go func() {
        fmt.Println("painter starting")
        for pCount := 0; pCount < totalPixels; pCount++ {
            pix := <- painter
            dc.SetRGB(pix.paint.r, pix.paint.g, pix.paint.b)
            dc.SetPixel(pix.x, pix.y)
        }
        fmt.Println("painter done")
    }()
    fmt.Println("waiting for worker threads to finish")
    for thread := 0; thread < parallelism; thread++ {
        num := <- done
        fmt.Printf("thread %d is done\n", num)
    }
    fmt.Println("saving")
    dc.SavePNG(*filePtr)
    fmt.Println("done")
}
