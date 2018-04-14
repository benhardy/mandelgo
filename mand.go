package main

import "github.com/fogleman/gg"
import "math"
import "fmt"
import "flag"
import "time"
import "runtime"

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
        color { 0.22, 0.00, 0.30 }, // dark chartreuse
        color { 0.50, 0.00, 0.93 }, // something bluish
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
    left, top, right, bottom int
}
func (a*area) Pixels() int {
    return  (a.right - a.left + 1) * (a.bottom - a.top +1)
}

func (a*area) String() string {
    return fmt.Sprintf("(%d-%d, %d-%d  = %d)", a.left, a.right, a.top, a.bottom, a.Pixels())
}

type Board struct {
    iMax int
    xCenter, yCenter, scale, aspect float64
    width, height int
    workQueue chan area
    doneQueue chan int
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

func (b* Board) walkPerimeter(rect area, thread int, job int) {
    //rectDesc := rect.String()
    //fmt.Printf("[%d:%d] walkPerimeter%s starting\n", thread, job, rectDesc)
    if rect.left > rect.right || rect.top > rect.bottom {
        //fmt.Printf("[%d:%d] walkPerimeter%s did nothing\n", thread, job, rectDesc)
        return
    }
    if rect.left == rect.right && rect.top == rect.bottom {
      //fmt.Printf("[%d:%d] walkPerimeter%s single pixel\n", thread, job, rectDesc)
        b.handlePixel(rect.left, rect.top)
        b.doneQueue<-1
        return
    }
    totalCount := 0
    escapeCount := 0

    if rect.right - rect.left < 8 || rect.bottom - rect.top < 8 { // eh, don't bother with recursive overhead with small squares
        for yp := rect.top; yp <= rect.bottom; yp++ {
            for xp := rect.left; xp <= rect.right; xp++ {
                escapeCount += b.handlePixel(xp, yp)
                totalCount++
            }
        }
        //fmt.Printf("[%d] walkPerimeter%s small  block %d pixels\n", thread, rectDesc, totalCount)
        b.doneQueue<-totalCount
        return
    }

    // draw top edge
    //fmt.Printf("%s walkPerimeter(%d, %d, %d, %d) doing top edge\n", indent, rect.left, rect.top, rect.right, rect.bottom)

    for xp := rect.left; xp <= rect.right; xp++ {
        escapeCount += b.handlePixel(xp, rect.top)
        totalCount++
    }
    // draw bottom edge
    if rect.bottom > rect.top {
        //fmt.Printf("%s walkPerimeter(%d, %d, %d, %d) doing bottom edge\n", indent, rect.left, rect.top, rect.right, rect.bottom)
        for xp := rect.left; xp <= rect.right; xp++ {
            escapeCount += b.handlePixel(xp, rect.bottom)
            totalCount++
        }
    }
    // draw left edge
    //fmt.Printf("%s walkPerimeter(%d, %d, %d, %d) doing left edge\n", indent, rect.left, rect.top, rect.right, rect.bottom)
    for yp := rect.top + 1; yp < rect.bottom; yp++ {
        escapeCount += b.handlePixel(rect.left, yp)
        totalCount++
    }
    // draw right edge
    if rect.right > rect.left {
        //fmt.Printf("%s walkPerimeter(%d, %d, %d, %d) doing right edge\n", indent, rect.left, rect.top, rect.right, rect.bottom)
        for yp := rect.top + 1; yp < rect.bottom; yp++ {
            escapeCount += b.handlePixel(rect.right, yp)
            totalCount++
        }
    }
    // no more drawing happens in this call, report results
    // we just drew the perimeter and it contained no
    // non-black pixels, so lets assume the whole thing is black
    if escapeCount == 0 {
        //fmt.Printf("[%d:%d] walkPerimeter%s is a black box vielding %d pixels\n", thread, job, rectDesc, rect.Pixels())
        //fmt.Printf("%s walkPerimeter(%d, %d, %d, %d) is a black box. avoiding that.\n", indent, rect.left, rect.top, rect.right, rect.bottom)
        b.doneQueue  <-rect.Pixels()
        return
    }
    //fmt.Printf("[%d:%d] walkPerimeter%s boundary provided %d pixels\n", thread, job, rectDesc, totalCount)
    //fmt.Printf("%s walkPerimeter(%d, %d, %d, %d) analyzed perimeter, %d/%d escaped\n", indent, rect.left, rect.top, rect.right, rect.bottom, escapeCount, totalCount)
    b.doneQueue<-totalCount
    xMid, yMid := (rect.left + rect.right) /2, (rect.top + rect.bottom)/2
    rect.left++
    rect.right--
    rect.top++
    rect.bottom--
    if rect.right - rect.left < 0 || rect.bottom - rect.top < 0 {
        //fmt.Printf("[%d:%d] walkPerimeter%s has no futher to go\n", thread, job, rectDesc)
        return
    }
    //fmt.Printf("[%d:%d] walkPerimeter%s subdividing\n", thread, job, rectDesc)
    b.workQueue<-area{rect.left, rect.top, xMid, yMid}
    if xMid < rect.right {
        b.workQueue<-area{xMid+1, rect.top, rect.right, yMid}
        if yMid < rect.bottom {
            b.workQueue<-area{xMid+1, yMid+1, rect.right, rect.bottom}
        }
    }
    if yMid < rect.bottom {
        b.workQueue<-area{rect.left, yMid+1, xMid, rect.bottom}
    }
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
        workQueue: make(chan area, *widthPtr * *heightPtr),
        doneQueue: make(chan int, *widthPtr * *heightPtr),
        palette: NewPalette(),
    }
    board.init()
    dc := gg.NewContext(board.width, board.height)

    go func() {
        fmt.Println("seeding work queue")
        board.workQueue <-area { 0, 0, board.width -1, board.height -1}
        fmt.Println("seeded work queue")
    }()
    cpus := runtime.NumCPU()
    threads := cpus // /4 +1
    // painter
    for t := 0; t<threads; t++ {
        fmt.Printf("starting thread %d\n", t)
        go func(id int) {
            jobCount := 0
            for {
                //fmt.Printf("[%d] thread waiting for a job\n", id)
                spot := <-board.workQueue
                if spot.left < 0 {
                    break
                }
                jobCount++
                //fmt.Printf("[%d] thread picked up job %d:%d %s\n", id, id, jobCount, &spot)
                board.walkPerimeter(spot, id, jobCount)
                //fmt.Printf("[%d] thread finished a job %d:%d %s\n", id, id, jobCount, &spot)
            }
        }(t)
    }
    fmt.Println("threads created, measuring completions")
    for totalRemaining := board.width * board.height; totalRemaining > 0; {
        doneBit := <-board.doneQueue
        totalRemaining -= doneBit
        fmt.Printf("\r%d pixels remaining   ", totalRemaining)
        //fmt.Printf("\r%.d pixels remaining", totalRemaining)
    }
    time.Sleep(1)
    fmt.Println("\nlooks like we are done, telling threads to quit")
    for t := 0; t<threads; t++ {
        board.workQueue<-area { left: -1 }
    }

    fmt.Println("\npreparing to save")
    for y := 0; y < board.height; y++ {
        for x :=0; x < board.width; x++ {
            col := board.grid[y][x]
            if col.r < 0.0 {
                col = color { 0.0, 0.0, 0.0 }
            }
            dc.SetRGB(col.r, col.g, col.b)
            dc.SetPixel(x, y)
        }
    }
    fmt.Println("saving")
    dc.SavePNG(*filePtr)
    fmt.Println("done")
}
