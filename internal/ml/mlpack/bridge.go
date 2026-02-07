package mlpack

// #cgo CXXFLAGS: -std=c++17 -stdlib=libc++
// #cgo CPPFLAGS: -I${SRCDIR} -I/usr/local/include -I/opt/homebrew/include
// #cgo darwin CXXFLAGS: -isysroot /Library/Developer/CommandLineTools/SDKs/MacOSX.sdk
// #cgo LDFLAGS: -L/usr/local/lib -L/opt/homebrew/lib -larmadillo -lstdc++ -lc++
// #include "anomaly_detector.hpp"
import "C"
import (
	"fmt"
	"unsafe"
)

// IsolationForest wraps the C++ Isolation Forest implementation
type IsolationForest struct {
	detector   unsafe.Pointer
	numTrees   int
	sampleSize int
}

// NewIsolationForest creates a new Isolation Forest detector
func NewIsolationForest(numTrees, sampleSize int) *IsolationForest {
	return &IsolationForest{
		detector:   C.create_isolation_forest(C.int(numTrees), C.int(sampleSize)),
		numTrees:   numTrees,
		sampleSize: sampleSize,
	}
}

// Fit trains the Isolation Forest on data
// data is a 2D slice where each row is a sample
func (iforest *IsolationForest) Fit(data [][]float64) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}

	rows := len(data)
	cols := len(data[0])

	// Flatten data to C array
	flatData := make([]float64, rows*cols)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			flatData[i*cols+j] = data[i][j]
		}
	}

	C.fit_isolation_forest(
		iforest.detector,
		(*C.double)(unsafe.Pointer(&flatData[0])),
		C.int(rows),
		C.int(cols),
	)

	return nil
}

// Score calculates anomaly scores for samples
// Returns scores where higher = more anomalous
func (iforest *IsolationForest) Score(samples [][]float64) ([]float64, error) {
	if len(samples) == 0 {
		return nil, fmt.Errorf("empty samples")
	}

	nSamples := len(samples)
	nFeatures := len(samples[0])

	// Flatten samples
	flatSamples := make([]float64, nSamples*nFeatures)
	for i := 0; i < nSamples; i++ {
		for j := 0; j < nFeatures; j++ {
			flatSamples[i*nFeatures+j] = samples[i][j]
		}
	}

	// Allocate scores array
	scores := make([]float64, nSamples)

	C.score_isolation_forest(
		iforest.detector,
		(*C.double)(unsafe.Pointer(&flatSamples[0])),
		C.int(nSamples),
		C.int(nFeatures),
		(*C.double)(unsafe.Pointer(&scores[0])),
	)

	return scores, nil
}

// Free releases the C++ resources
func (iforest *IsolationForest) Free() {
	if iforest.detector != nil {
		C.free_isolation_forest(iforest.detector)
		iforest.detector = nil
	}
}

// LOF wraps the C++ Local Outlier Factor implementation
type LOF struct {
	detector unsafe.Pointer
	k        int
}

// NewLOF creates a new LOF detector
func NewLOF(k int) *LOF {
	return &LOF{
		detector: C.create_lof(C.int(k)),
		k:        k,
	}
}

// Compute calculates LOF scores for data
func (lof *LOF) Compute(data [][]float64) ([]float64, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	rows := len(data)
	cols := len(data[0])

	flatData := make([]float64, rows*cols)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			flatData[i*cols+j] = data[i][j]
		}
	}

	scores := make([]float64, rows)

	C.compute_lof(
		lof.detector,
		(*C.double)(unsafe.Pointer(&flatData[0])),
		C.int(rows),
		C.int(cols),
		(*C.double)(unsafe.Pointer(&scores[0])),
	)

	return scores, nil
}

// Free releases the C++ resources
func (lof *LOF) Free() {
	if lof.detector != nil {
		C.free_lof(lof.detector)
		lof.detector = nil
	}
}

// DBSCANCluster performs DBSCAN clustering
func DBSCANCluster(data [][]float64, eps float64, minPts int) ([]int, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	rows := len(data)
	cols := len(data[0])

	flatData := make([]float64, rows*cols)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			flatData[i*cols+j] = data[i][j]
		}
	}

	labels := make([]int, rows)

	C.dbscan_cluster(
		(*C.double)(unsafe.Pointer(&flatData[0])),
		C.int(rows),
		C.int(cols),
		C.double(eps),
		C.int(minPts),
		(*C.int)(unsafe.Pointer(&labels[0])),
	)

	return labels, nil
}
