package main

import (
	"fmt"
	"log"
	"math"

	"go-hep.org/x/hep/fmom"
	"go-hep.org/x/hep/groot"
	"go-hep.org/x/hep/groot/rtree"

	"github.com/rmadar/go-lorentz-vector/lv"
	"github.com/rmadar/go-topquark-reco/sonn"
)

func main() {

	sonn.InitLogs()
	defer sonn.StopLogs()

	// Open the test ROOT file
	f, err := groot.Open("../testdata/data.root")
	if err != nil {
		log.Fatalf("could not open ROOT file: %+v", err)
	}
	defer f.Close()

	// Get the TTree
	o, err := f.Get("nominal")
	if err != nil {
		log.Fatalf("could not retrieve ROOT tree: %+v", err)
	}
	t := o.(rtree.Tree)

	// Get variables to read
	var (
		nBad      = 0
		evtNum    int64
		lepPt     []float32
		lepEta    []float32
		lepPhi    []float32
		lepPid    []int32
		jetPt     []float32
		jetE      []float32
		jetEta    []float32
		jetPhi    []float32
		jetMV2c10 []float32
		nBjets    int32
		metMet    float32
		metPhi    float32

		rvars = []rtree.ReadVar{
			{Name: "eventNumber", Value: &evtNum},
			{Name: "d_lep_pt", Value: &lepPt},
			{Name: "d_lep_eta", Value: &lepEta},
			{Name: "d_lep_phi", Value: &lepPhi},
			{Name: "d_lep_pid", Value: &lepPid},
			{Name: "d_jet_pt", Value: &jetPt},
			{Name: "d_jet_e", Value: &jetE},
			{Name: "d_jet_eta", Value: &jetEta},
			{Name: "d_jet_phi", Value: &jetPhi},
			{Name: "d_jet_mv2c10", Value: &jetMV2c10},
			{Name: "d_nbjet", Value: &nBjets},
			{Name: "d_met_met", Value: &metMet},
			{Name: "d_met_phi", Value: &metPhi},
		}
	)

	// Get the TTree reader
	r, err := rtree.NewReader(t, rvars, rtree.WithRange(0, 100))
	if err != nil {
		log.Fatalf("could not create tree reader: %+v", err)
	}
	defer r.Close()

	// Load smearing histograms
	sonn.SetupSmearingFile("../testdata/smearingHistos.root")

	sh, err := sonn.NewSmearingHistos("../testdata/smearingHistos.root", 1234)
	if err != nil {
		log.Fatalf("could not load smearing histos: %+v", err)
	}

	// Event loop
	err = r.Read(func(ctx rtree.RCtx) error {

		// Prepare leptons four vectors
		var lP, lbarP lv.FourVec
		var lId, lbarId int
		for i, id := range lepPid {
			if id > 0 {
				lbarId = int(id)
				lbarP = lv.NewFourVecPtEtaPhiM(float64(lepPt[i]), float64(lepEta[i]), float64(lepPhi[i]), 0.0)
			} else {
				lId = int(id)
				lP = lv.NewFourVecPtEtaPhiM(float64(lepPt[i]), float64(lepEta[i]), float64(lepPhi[i]), 0.0)
			}
		}

		// Prepare jet four vectors and b-tagg info based on the 2 leading jets
		var j1P, j2P lv.FourVec
		j1P = lv.NewFourVecPtEtaPhiE(float64(jetPt[0]), float64(jetEta[0]), float64(jetPhi[1]), float64(jetE[0]))
		j2P = lv.NewFourVecPtEtaPhiE(float64(jetPt[1]), float64(jetEta[1]), float64(jetPhi[1]), float64(jetE[1]))
		var (
			j1b = jetMV2c10[0] > 0.691
			j2b = jetMV2c10[1] > 0.691
		)

		// Prepare missing transverse energy component
		sin, cos := math.Sincos(float64(metPhi))
		Etx := float64(metMet) * cos
		Ety := float64(metMet) * sin

		// Call the Sonnenschein reconstruction
		reco := sonn.RecoTops
		if false {
			reco = sonn.Sonnenschein
		}
		tops := reco(
			fmomP4from(lP), fmomP4from(lbarP), lId, lbarId,
			fmomP4from(j1P), fmomP4from(j2P), j1b, j2b,
			Etx, Ety, sh,
		)
		var xtops []fmom.PxPyPzE
		if true {
			xtops = sonn.Sonnenschein(
				fmomP4from(lP), fmomP4from(lbarP), lId, lbarId,
				fmomP4from(j1P), fmomP4from(j2P), j1b, j2b,
				Etx, Ety, sh,
			)
		}

		bad := false
		// Keep track of not reconstructed events
		if len(tops) == 0 || isBad(tops[0]) || isBad(tops[1]) {
			nBad++
			bad = true
		}

		// Print some information
		fmt.Printf("Entry %d: n-bad=%d\n", ctx.Entry, nBad)
		fmt.Printf("   - Evt number   %v\n", evtNum)
		fmt.Printf("   - N[b-jets]    %v\n", nBjets)
		fmt.Printf("   - final state  %v\n", lepPid)
		fmt.Printf("   - P4[l]        %v\n", fmomP4from(lP))
		fmt.Printf("   - P4[lbar]     %v\n", fmomP4from(lbarP))
		fmt.Printf("   - P4[top]      %v\n", tops[0])
		fmt.Printf("   - P4[anti-top] %v\n", tops[1])
		fmt.Printf("   + bad=         %v\n", bad)
		if len(xtops) > 0 {
			fmt.Printf("   + P4[top]      %v\n", xtops[0])
			fmt.Printf("   + P4[anti-top] %v\n", xtops[1])
		}
		fmt.Printf("\n")

		return nil
	})

	fmt.Printf("Number of events w/o reconstruction: %v\n\n", nBad)

	if err != nil {
		log.Fatalf("could not process tree: %+v", err)
	}

}

// Helper function to get a fmom.P4 from lv.FourVec
func fmomP4from(fv lv.FourVec) fmom.P4 {
	res := fmom.NewPxPyPzE(fv.Px(), fv.Py(), fv.Pz(), fv.E())
	return &res
}

// Helper function checking of the P4[top] makes sense
func isBad(t fmom.PxPyPzE) bool {
	isDefault := t.Px() == 10000. && t.Py() == 10000. && t.Pz() == 10000.
	isEmpty := t.Px() == 0. && t.Py() == 0. && t.Pz() == 0.
	return isDefault || isEmpty
}
