# mandelgo
I am currently learning Go, so this is my way of saying Hello World as a Mandelbrot set generator in Go. It has some interesting optimizations.

Please bear in mind that this code is currently very messy and was written for toodling around at home experimentally.

BUUUUT someone asked me to share it so here it is. Now I have to clean it up...

## How it works

This program is built around the assumption that if all the pixels along the boundary of any rectangle in the complex
plane never escape the Mandelbrot Set (i.e. the black bits), then neither will any pixel inside that rectangle, so
we can skip analyzing those entirely.

If a perimeter does contain *any* escaping coordinates, then we break that rectangle into smaller ones and analyze those.
When the rectangles get small enough we just analyze everything in them.

The implementation is not straight up recursive though. Each rectangle to be analyzed is fed as a job into a work queue. A 
Goroutine is running for each CPU, which picks up a job off the work queue and calls `walkPerimeter` with it. This 
function figures out what to do based on what it discovers about the perimeter. If it determines that there is more work
to be done, instead of recursing, it feeds smaller jobs back into the job queue, so any Goroutine that's available could
pick one up.

Pixels are rendered to a big array, initially. This is safe to be lock-free. The gg library which we use to render the resulting
PNG files is _not_ concurrency safe, so we leave usage of that for when it's time to save the file. 

## Get the code and build it
```
# This has one dependency, gg
go get github.com/fogleman/gg github.com/benhardy/mandelgo
if [ "$GOPATH" != "" ] ; then cd $GOPATH; else cd ~/go/src; fi
cd github.com/benhardy/mandelgo
go build
```

# Usage
Usage is via command line parameters, see code for details, but here's an example:

`./mandelgo -width=2000 -height=1500 -iterations=3000 -real=-0.7451 -imag=0.139 -scale=0.002 -save=hello.png`

