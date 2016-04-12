package plotutil


import (
	"github.com/skiesel/plot/plotter"
	"fmt"
	"sort"
)

func NewPaddedErrorPoints(f func([]float64) (c, l, h float64), lines ...[]plotter.XYer) ([]*ErrorPoints, error) {
	errPts := make([]*ErrorPoints, len(lines))
	for i, line := range lines {
		pts, err := newPaddedErrorPoints(f, i, len(lines), line...)
		if err != nil {
			return nil, err
		}
		errPts[i] = pts
	}
	return errPts, nil
}

func newPaddedErrorPoints(f func([]float64) (c, l, h float64), cur, total int, points ...plotter.XYer) (*ErrorPoints, error) {
	defaultNumErrBars := 4

	errBars := &ErrorPoints{
		XYs:     make(plotter.XYs, defaultNumErrBars),
		XErrors: make(plotter.XErrors, defaultNumErrBars),
		YErrors: make(plotter.YErrors, defaultNumErrBars),
	}

	return errBars, nil
}

type Float64Range struct {
	min, max float64
}

type PointGenerator struct {
	// this function takes in an x value
	// and returns an array of y values at that
	// x point
	generator func(float64) []float64

	// the min and max x value this generator
	// is applicable
	pointRange Float64Range
}

type PointsAndErrorPoints struct {
	points *plotter.XYs
	errorPoints *ErrorPoints
}

func NewErrorPointsXSpaced(f func([]float64) (c, l, h float64),
														howManyPoints, howManyErrorBars int64,
														dataRange Float64Range,
														pointsGenerators []PointGenerator) (*[]PointsAndErrorPoints, error) {
	if howManyPoints < 2 {
		panic("NewErrorPointsXSpaced: only 2 or more points can be used")
	}

	dataMin := dataRange.min
	dataMax := dataRange.max

	ptsNerrs := make([]PointsAndErrorPoints, len(pointsGenerators))

	//Figure out where the data points go
	xIncr := (dataMax - dataMin) / (float64(howManyPoints) - 1.0)
	for curGeneratorIndex, pointsGenerator := range pointsGenerators {

		//create and instantiate the PointsAndErrorPoints object
		points := make(plotter.XYs, howManyPoints)
		ptsNerrs[curGeneratorIndex].points = &points

		errorPoints := ErrorPoints{
			XYs:     make(plotter.XYs, howManyErrorBars),
			XErrors: make(plotter.XErrors, howManyErrorBars),
			YErrors: make(plotter.YErrors, howManyErrorBars),
		}
		ptsNerrs[curGeneratorIndex].errorPoints = &errorPoints

		//rename a few things for ease of reading the code
		genRangeMin := pointsGenerator.pointRange.min
		genRangeMax := pointsGenerator.pointRange.max
		generator := pointsGenerator.generator


		//start by generating the data points that will define the line
		ptsAdded := 0
		for curX := dataMin; ptsAdded < points.Len() && curX <= dataMax + xIncr; curX += xIncr {

			//if the current x value is outside of the generator range skip it
			if curX < genRangeMin || curX > genRangeMax {
				continue
			}

			//make sure that the x value we're going to use is okay
			if err := plotter.CheckFloats(curX); err != nil {
				return nil, err
			}

			//generate the set of y values at this given x value
			ys := generator(curX)
			//make sure that all the y values returned are okay
			if err := plotter.CheckFloats(ys...); err != nil {
				return nil, err
			}

			//set the x value for this point
			points[ptsAdded].X = curX
			//set the y value for this point as returned by this function
			//we only want the points so ignore the range-type of data also returned
			points[ptsAdded].Y, _, _ = f(ys)
			//make sure that the returned value is okay
			if err := plotter.CheckFloats(points[ptsAdded].Y); err != nil {
				return nil, err
			}
			//We need to keep track of how many points we're actually adding
			//because not all the x's will fall within the generators range
			//and we might have to shrink the array to avoid problems when plotting
			ptsAdded++
		}
		points = points[0:ptsAdded]


		//Now figure out where the error bars go
		xIncr = (dataMax - dataMin) / float64(howManyErrorBars)
		//we want to space out the error bars so they don't all overlap
		xOffset := (xIncr / 4.0) + float64(curGeneratorIndex) * (xIncr / 2.0) / float64(len(pointsGenerators))
		numErrsAdded := 0
		for curX := dataMin; numErrsAdded < errorPoints.XYs.Len() && curX <= dataMax + xIncr; curX += xIncr {

			x := curX + xOffset

			if curX < genRangeMin || curX > genRangeMax || x < genRangeMin || x > genRangeMax {
				continue
			}

			if err := plotter.CheckFloats(x); err != nil {
				return nil, err
			}

			ys := generator(x)
			if err := plotter.CheckFloats(ys...); err != nil {
				return nil, err
			}
			
			errorPoints.XYs[numErrsAdded].X = x
			errorPoints.XErrors[numErrsAdded].Low = 0.0
			errorPoints.XErrors[numErrsAdded].High = 0.0

			y, low, high := f(ys)

			errorPoints.XYs[numErrsAdded].Y = y
			errorPoints.YErrors[numErrsAdded].Low = low
			errorPoints.YErrors[numErrsAdded].High = high

			if err := plotter.CheckFloats(errorPoints.YErrors[numErrsAdded].Low, errorPoints.YErrors[numErrsAdded].High); err != nil {
				return nil, err
			}
			numErrsAdded++
		}
		ptsNerrs[curGeneratorIndex].errorPoints.XYs = errorPoints.XYs[0:numErrsAdded]
		ptsNerrs[curGeneratorIndex].errorPoints.XErrors = errorPoints.XErrors[0:numErrsAdded]
		ptsNerrs[curGeneratorIndex].errorPoints.YErrors = errorPoints.YErrors[0:numErrsAdded]
	}

	return &ptsNerrs, nil
}

func StepFunction(data map[float64][]float64) func(float64)[]float64 {
	sorted := make([]float64, len(data))
	i := 0
	for key, _ := range data {
		sorted[i] = key
		i++
	}

	sort.Float64s(sorted)

	return func(x float64)[]float64 {
		curPoint := 0
		for ; curPoint < len(sorted) && sorted[curPoint] <= x; curPoint++ {
		}
		if curPoint >= len(sorted) {
			curPoint = len(sorted) - 1
		}

		if curPoint == 0 && sorted[curPoint] > x {
			fmt.Printf("values at x=%f are undefined, earliest point in step function defined at x=%f\n", x, sorted[curPoint])
			panic("undefined value")
		}

		return data[sorted[curPoint]]
	}
}

//This function assumes that the data across []float64 retrieved from the map are
//ordered such that data[key][n] and data[anotherkey][n] are those to be interpolated between
func LinearInterpolationFunction(data map[float64][]float64) func(float64)[]float64 {

	sorted := make([]float64, len(data))
	size := -1
	i := 0
	for key, slice := range data {
		sorted[i] = key
		if i == 0 {
			size = len(slice)
		} else if size != len(slice) {
			panic("LinearInterpolationFunction -- mismatched []float64 sizes, they must all be equal to interpolate")
		}
		i++
	}

	sort.Float64s(sorted)

	return func(x float64)[]float64 {
		point2 := 0
		for ; point2 < len(sorted) && sorted[point2] < x; point2++ {
		}
		if point2 >= len(sorted) || point2 == 0 {
			fmt.Printf("value x=%f exceeds defined range x=%f -> x=%f\n", x, sorted[0], sorted[len(sorted)-1])
			panic("undefined value")
		}

		point1 := point2 - 1
		values := make([]float64, len(data[sorted[point1]]))
		for i := range values {
			values[i] = (data[sorted[point1]][i] + data[sorted[point2]][i]) / 2.0
		}

		return values
	}
}

func NewErrorPointsSpaced(f func([]float64) (c, l, h float64),
						pointSet, totalNumberOfPointSets,
						howManyPoints, howManyErrorBars int64,
						minX, maxX float64,
						pointsGenerator func(float64) []float64,
						minRange, maxRange float64) (*plotter.XYs, *ErrorPoints, error) {

	if howManyPoints < 2 {
		panic("NewErrorPointsXSpaced: only 2 or more points can be used")
	}

	points := make(plotter.XYs, howManyPoints)

	errorBars := &ErrorPoints{
		XYs:     make(plotter.XYs, howManyErrorBars),
		XErrors: make(plotter.XErrors, howManyErrorBars),
		YErrors: make(plotter.YErrors, howManyErrorBars),
	}

	//generate the points first
	xIncr := (maxX - minX) / (float64(howManyPoints) - 1.0)
	i := 0
	for curX := minX; i < points.Len() && curX <= maxX + xIncr; curX += xIncr {

		if curX < minRange || curX > maxRange {
			continue
		}

		if err := plotter.CheckFloats(curX); err != nil {
			return nil, nil, err
		}

		ys := pointsGenerator(curX)

		for j := 0; j < len(ys); j++ {
			if err := plotter.CheckFloats(ys[j]); err != nil {
				return nil, nil, err
			}
		}

		points[i].X = curX
		points[i].Y, _, _ = f(ys)
		if err := plotter.CheckFloats(points[i].Y); err != nil {
			return nil, nil, err
		}
		i++
	}

	points = points[0:i]

	//generate the error bars
	xIncr = (maxX - minX) / float64(howManyErrorBars)
	xOffset := (xIncr / 4.0) + float64(pointSet) * (xIncr / 2.0) / float64(totalNumberOfPointSets)
	i = 0
	for curX := minX; i < errorBars.XYs.Len() && curX <= maxX + xIncr; curX += xIncr {

		x := curX + xOffset

		if curX < minRange || curX > maxRange ||x < minRange || x > maxRange {
			continue
		}

		if err := plotter.CheckFloats(x); err != nil {
			return nil, nil, err
		}

		ys := pointsGenerator(x)

		for j := 0; j < len(ys); j++ {
			if err := plotter.CheckFloats(ys[j]); err != nil {
				return nil, nil, err
			}
		}
		
		errorBars.XYs[i].X, errorBars.XErrors[i].Low, errorBars.XErrors[i].High = x, 0.0, 0.0
		errorBars.XYs[i].Y, errorBars.YErrors[i].Low, errorBars.YErrors[i].High = f(ys)
		if err := plotter.CheckFloats(errorBars.YErrors[i].Low, errorBars.YErrors[i].High); err != nil {
			return nil, nil, err
		}
		i++
	}
	errorBars.XYs = errorBars.XYs[0:i]
	errorBars.XErrors = errorBars.XErrors[0:i]
	errorBars.YErrors = errorBars.YErrors[0:i]


	return &points, errorBars, nil
}