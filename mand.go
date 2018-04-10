package main

import "github.com/fogleman/gg"
import "math"
import "fmt"
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

type Palette struct {
    chooser func(iterations float64) color
    black color
}
func NewPalette() *Palette {
    p := Palette{}
    palette := []color{
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
    p.chooser = func(iterations float64) color {
        palPos := math.Log(iterations)
        palBase := int(palPos)
        palFirst := palBase % len(palette)
        palNext := (palBase + 1) % len(palette)
        palIndex := math.Floor(palPos)
        fraction := palPos - palIndex
        return palette[palFirst].smooth(palette[palNext], fraction)
    }
    p.black = color { 0.0, 0.0, 0.0 }
    return &p
}

type pixel struct {
    x int
    y int
    paint color
}
type area struct {
    left, top, width, height int
    paint color
}
func (a *area) numPixels() int {
    return a.width * a.height
}

type Board struct {
    iMax int
    xCenter, yCenter, scale, aspect float64
    width, height int
    workQueue chan area
    paintQueue chan pixel
    palette *Palette
    grid [][]color
}

func (b *Board) init() {
    b.grid = make([][]color, b.height)
    for y := 0; y < b.height; y++ {
        b.grid[y] = make([]color, b.width)
        for x := 0; x < b.width; x++ {
            b.grid[y][x] = color { -1.0, -1.0, -1.0 }
        }
    }
}
func (b *Board) setPixel(x int, y int, col color) {
    if b.grid[y][x].r < 0 {
        b.grid[y][x] = col
        //fmt.Printf("setPixel(%d, %d) OK\n", x, y)
    } else {
        b.grid[y][x] = color { 1.0, 0.0, 0.0 }
        //fmt.Printf("setPixel(%d, %d) has been here before! wtf!\n", x, y)
    }
}
func (b *Board) iterate(xc float64, yc float64) (bool, float64) {
    xp := 0.0
    yp := 0.0
    var i int
    escaped := false
    xp2 := 0.0
    yp2 := 0.0
    iMax := b.iMax
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
    if !escaped {
        return false, 0
    }
    logZn := math.Log(xp2 + yp2) / 2.0
    nu := math.Log(logZn / math.Log(2) ) / math.Log(2)
    iSmooth := float64(i) - nu + 1
    return true, iSmooth
}
func (b* Board) coordinates(xPixel int, yPixel int) (float64, float64) {
    xc := b.xCenter + b.scale * (float64(xPixel) / float64(b.width) - 0.5)
    yc := b.yCenter - b.aspect * b.scale * (float64(yPixel) / float64(b.height) - 0.5)
    return xc, yc
}
func (b *Board) handlePixel(x int, y int) int {
    xc, yc := b.coordinates(x, y)
    escaped, iSmooth := b.iterate(xc, yc)
    if escaped {
        //spot := pixel { x, y, b.palette.chooser(iSmooth) }
        //b.paintQueue <-spot
        b.setPixel(x, y, b.palette.chooser(iSmooth))
        return 1
    }
    b.setPixel(x, y, b.palette.black)
    //spot := pixel { x, y, b.palette.black }
    //b.paintQueue <-spot
    return 0
}

func (b* Board) walkPerimeter(xLeft, yTop, xRight, yBottom, depth int) int {
    //indentFmt := fmt.Sprintf("%%%ds", depth*4)
    //indent := fmt.Sprintf(indentFmt, " ")
    if xLeft > xRight || yTop > yBottom {
        //fmt.Printf("%s walkPerimeter(%d, %d, %d, %d) has nothing to do\n", indent, xLeft, yTop, xRight, yBottom)
        return 0
    }
    if xLeft == xRight && yTop == yBottom {
        //fmt.Printf("%s walkPerimeter(%d, %d, %d, %d) doing single pixel\n", indent, xLeft, yTop, xRight, yBottom)
        return b.handlePixel(xLeft, yTop)
    }
    totalCount := 0
    escapeCount := 0
    if xRight - xLeft < 8 || yBottom - yTop < 8 { // eh, don't bother with recursive overhead with small squares
        totalCount := 0
        for yp := yTop; yp <= yBottom; yp++ {
            for xp := xLeft; xp <= xRight; xp++ {
                escapeCount += b.handlePixel(xp, yp)
                totalCount++
            }
        }
        return totalCount
    }
    // draw top edge
    //fmt.Printf("%s walkPerimeter(%d, %d, %d, %d) doing top edge\n", indent, xLeft, yTop, xRight, yBottom)

    for xp := xLeft; xp <= xRight; xp++ {
        escapeCount += b.handlePixel(xp, yTop)
        totalCount++
    }
    // draw bottom edge
    if yBottom > yTop {
        //fmt.Printf("%s walkPerimeter(%d, %d, %d, %d) doing bottom edge\n", indent, xLeft, yTop, xRight, yBottom)
        for xp := xLeft; xp <= xRight; xp++ {
            escapeCount += b.handlePixel(xp, yBottom)
            totalCount++
        }
    }
    // draw left edge
    //fmt.Printf("%s walkPerimeter(%d, %d, %d, %d) doing left edge\n", indent, xLeft, yTop, xRight, yBottom)
    for yp := yTop + 1; yp < yBottom; yp++ {
        escapeCount += b.handlePixel(xLeft, yp)
        totalCount++
    }
    // draw right edge
    if xRight > xLeft {
        //fmt.Printf("%s walkPerimeter(%d, %d, %d, %d) doing right edge\n", indent, xLeft, yTop, xRight, yBottom)
        for yp := yTop + 1; yp < yBottom; yp++ {
            escapeCount += b.handlePixel(xRight, yp)
            totalCount++
        }
    }
    // we just drew the perimeter and it contained no
    // non-black pixels, so lets assume the whole thing is black
    if escapeCount == 0 {
        //fmt.Printf("%s walkPerimeter(%d, %d, %d, %d) is a black box. avoiding that.\n", indent, xLeft, yTop, xRight, yBottom)
        return 0
    }
    //fmt.Printf("%s walkPerimeter(%d, %d, %d, %d) analyzed perimeter, %d/%d escaped\n", indent, xLeft, yTop, xRight, yBottom, escapeCount, totalCount)
    xMid, yMid := (xLeft + xRight) /2, (yTop + yBottom)/2
    depth++
    xLeft++
    xRight--
    yTop++
    yBottom--
    if xRight - xLeft <= 0 || yBottom - yTop <= 0 {
        return 0
    }
    subCount := b.walkPerimeter(xLeft, yTop, xMid, yMid, depth) +
        b.walkPerimeter(xMid+1, yTop, xRight, yMid, depth) +
        b.walkPerimeter(xLeft, yMid+1, xMid, yBottom, depth) +
        b.walkPerimeter(xMid+1, yMid+1, xRight, yBottom, depth)
    //fmt.Printf("%s walkPerimeter(%d, %d, %d, %d) children yielded %d pixels.\n", indent, xLeft, yTop, xRight, yBottom, subCount)
    return subCount
}

func main() {
    widthPtr   := flag.Int("width", 2000, "image width in pixels")
    heightPtr  := flag.Int("height", 1200, "image height in pixels")
    iMaxPtr    := flag.Int("iterations", 3000, "maximum number of iterations")
    xCenterPtr := flag.Float64("real", -0.745, "real component of image center")
    yCenterPtr := flag.Float64("imag", 0.149,  "imaginary component of image center")
    scalePtr   := flag.Float64("scale", 0.03, "width of view window in complex plane")
    filePtr    := flag.String("save", "out.png", "name of file to save image to")
    flag.Parse()
    board := Board {
        width: *widthPtr,
        height: *heightPtr,
        iMax: *iMaxPtr,
        xCenter: *xCenterPtr,
        yCenter:  *yCenterPtr,
        scale: *scalePtr,
        aspect : float64(*heightPtr) / float64(*widthPtr),
        workQueue: make(chan area),
        paintQueue: make(chan pixel),
        palette: NewPalette(),
    }
    board.init()
    dc := gg.NewContext(board.width, board.height)
    // painter
    fmt.Println("rendering")
    board.walkPerimeter(0, 0, board.width-1, board.height-1, 0)
    fmt.Println("preparing to save")
    for y := 0; y < board.height; y++ {
        for x :=0; x < board.width; x++ {
            col := board.grid[y][x]
            if col.r < 0.0 {
                col = color { 0.0, 0.3, 0.0 }
            }
            dc.SetRGB(col.r, col.g, col.b)
            dc.SetPixel(x, y)
        }
    }
    fmt.Println("saving")
    dc.SavePNG(*filePtr)
    fmt.Println("done")
}
