// implementation of https://eprint.iacr.org/2013/879.pdf

package proof

import (
	"fmt"
	"math/big"

	"github.com/arnaucube/go-snark/circuit"
)

// PinocchioSetup is Pinocchio system setup structure
type PinocchioSetup struct {
	Toxic struct {
		T      *big.Int // trusted setup secret
		Ka     *big.Int // prover
		Kb     *big.Int // prover
		Kc     *big.Int // prover
		Kbeta  *big.Int
		Kgamma *big.Int
		RhoA   *big.Int
		RhoB   *big.Int
		RhoC   *big.Int
	} `json:"-"`

	// public
	G1T [][3]*big.Int    // t encrypted in G1 curve, G1T == Pk.H
	G2T [][3][2]*big.Int // t encrypted in G2 curve
	Pk  struct {         // Proving Key pk:=(pkA, pkB, pkC, pkH)
		A  [][3]*big.Int
		B  [][3][2]*big.Int
		C  [][3]*big.Int
		Kp [][3]*big.Int
		Ap [][3]*big.Int
		Bp [][3]*big.Int
		Cp [][3]*big.Int
		Z  []*big.Int
	}
	Vk struct {
		Vka   [3][2]*big.Int
		Vkb   [3]*big.Int
		Vkc   [3][2]*big.Int
		IC    [][3]*big.Int
		G1Kbg [3]*big.Int    // g1 * Kbeta * Kgamma
		G2Kbg [3][2]*big.Int // g2 * Kbeta * Kgamma
		G2Kg  [3][2]*big.Int // g2 * Kgamma
		Vkz   [3][2]*big.Int
	}
}

// PinocchioProof is Pinocchio proof structure
type PinocchioProof struct {
	PiA  [3]*big.Int
	PiAp [3]*big.Int
	PiB  [3][2]*big.Int
	PiBp [3]*big.Int
	PiC  [3]*big.Int
	PiCp [3]*big.Int
	PiH  [3]*big.Int
	PiKp [3]*big.Int
}

// Z is ...
func (setup *PinocchioSetup) Z() []*big.Int {
	return setup.Pk.Z
}

// Init setups the trusted setup from a compiled circuit
func (setup *PinocchioSetup) Init(cir *circuit.Circuit, alphas, betas, gammas [][]*big.Int) error {
	var err error

	setup.Toxic.T, err = Utils.FqR.Rand()
	if err != nil {
		return err
	}

	setup.Toxic.Ka, err = Utils.FqR.Rand()
	if err != nil {
		return err
	}
	setup.Toxic.Kb, err = Utils.FqR.Rand()
	if err != nil {
		return err
	}
	setup.Toxic.Kc, err = Utils.FqR.Rand()
	if err != nil {
		return err
	}

	setup.Toxic.Kbeta, err = Utils.FqR.Rand()
	if err != nil {
		return err
	}
	setup.Toxic.Kgamma, err = Utils.FqR.Rand()
	if err != nil {
		return err
	}
	kbg := Utils.FqR.Mul(setup.Toxic.Kbeta, setup.Toxic.Kgamma)

	setup.Toxic.RhoA, err = Utils.FqR.Rand()
	if err != nil {
		return err
	}
	setup.Toxic.RhoB, err = Utils.FqR.Rand()
	if err != nil {
		return err
	}
	setup.Toxic.RhoC = Utils.FqR.Mul(setup.Toxic.RhoA, setup.Toxic.RhoB)

	setup.Vk.Vka = Utils.Bn.G2.MulScalar(Utils.Bn.G2.G, setup.Toxic.Ka)
	setup.Vk.Vkb = Utils.Bn.G1.MulScalar(Utils.Bn.G1.G, setup.Toxic.Kb)
	setup.Vk.Vkc = Utils.Bn.G2.MulScalar(Utils.Bn.G2.G, setup.Toxic.Kc)

	setup.Vk.G1Kbg = Utils.Bn.G1.MulScalar(Utils.Bn.G1.G, kbg)
	setup.Vk.G2Kbg = Utils.Bn.G2.MulScalar(Utils.Bn.G2.G, kbg)
	setup.Vk.G2Kg = Utils.Bn.G2.MulScalar(Utils.Bn.G2.G, setup.Toxic.Kgamma)

	for i := 0; i < len(cir.Signals); i++ {
		at := Utils.PF.Eval(alphas[i], setup.Toxic.T)
		rhoAat := Utils.FqR.Mul(setup.Toxic.RhoA, at)
		a := Utils.Bn.G1.MulScalar(Utils.Bn.G1.G, rhoAat)
		setup.Pk.A = append(setup.Pk.A, a)
		if i <= cir.NPublic {
			setup.Vk.IC = append(setup.Vk.IC, a)
		}

		bt := Utils.PF.Eval(betas[i], setup.Toxic.T)
		rhoBbt := Utils.FqR.Mul(setup.Toxic.RhoB, bt)
		bg1 := Utils.Bn.G1.MulScalar(Utils.Bn.G1.G, rhoBbt)
		bg2 := Utils.Bn.G2.MulScalar(Utils.Bn.G2.G, rhoBbt)
		setup.Pk.B = append(setup.Pk.B, bg2)

		ct := Utils.PF.Eval(gammas[i], setup.Toxic.T)
		rhoCct := Utils.FqR.Mul(setup.Toxic.RhoC, ct)
		c := Utils.Bn.G1.MulScalar(Utils.Bn.G1.G, rhoCct)
		setup.Pk.C = append(setup.Pk.C, c)

		kt := Utils.FqR.Add(Utils.FqR.Add(rhoAat, rhoBbt), rhoCct)
		k := Utils.Bn.G1.Affine(Utils.Bn.G1.MulScalar(Utils.Bn.G1.G, kt))

		ktest := Utils.Bn.G1.Affine(Utils.Bn.G1.Add(Utils.Bn.G1.Add(a, bg1), c))
		if !Utils.Bn.Fq2.Equal(k, ktest) {
			return err
		}

		setup.Pk.Ap = append(setup.Pk.Ap, Utils.Bn.G1.MulScalar(a, setup.Toxic.Ka))
		setup.Pk.Bp = append(setup.Pk.Bp, Utils.Bn.G1.MulScalar(bg1, setup.Toxic.Kb))
		setup.Pk.Cp = append(setup.Pk.Cp, Utils.Bn.G1.MulScalar(c, setup.Toxic.Kc))

		kk := Utils.Bn.G1.MulScalar(Utils.Bn.G1.G, kt)
		setup.Pk.Kp = append(setup.Pk.Kp, Utils.Bn.G1.MulScalar(kk, setup.Toxic.Kbeta))
	}

	zpol := []*big.Int{big.NewInt(int64(1))}
	for i := 1; i < len(alphas)-1; i++ {
		zpol = Utils.PF.Mul(
			zpol,
			[]*big.Int{
				Utils.FqR.Neg(big.NewInt(int64(i))),
				big.NewInt(int64(1)),
			})
	}
	setup.Pk.Z = zpol

	zt := Utils.PF.Eval(zpol, setup.Toxic.T)
	rhoCzt := Utils.FqR.Mul(setup.Toxic.RhoC, zt)
	setup.Vk.Vkz = Utils.Bn.G2.MulScalar(Utils.Bn.G2.G, rhoCzt)

	var gt1 [][3]*big.Int
	gt1 = append(gt1, Utils.Bn.G1.G)
	tEncr := setup.Toxic.T
	for i := 1; i < len(zpol); i++ {
		gt1 = append(gt1, Utils.Bn.G1.MulScalar(Utils.Bn.G1.G, tEncr))
		tEncr = Utils.FqR.Mul(tEncr, setup.Toxic.T)
	}
	setup.G1T = gt1

	return nil
}

// Generate generates Pinocchio proof
func (setup *PinocchioSetup) Generate(cir *circuit.Circuit, w []*big.Int, px []*big.Int) (Proof, error) {
	proof := &PinocchioProof{}
	proof.PiA = [3]*big.Int{Utils.Bn.G1.F.Zero(), Utils.Bn.G1.F.Zero(), Utils.Bn.G1.F.Zero()}
	proof.PiAp = [3]*big.Int{Utils.Bn.G1.F.Zero(), Utils.Bn.G1.F.Zero(), Utils.Bn.G1.F.Zero()}
	proof.PiB = Utils.Bn.Fq6.Zero()
	proof.PiBp = [3]*big.Int{Utils.Bn.G1.F.Zero(), Utils.Bn.G1.F.Zero(), Utils.Bn.G1.F.Zero()}
	proof.PiC = [3]*big.Int{Utils.Bn.G1.F.Zero(), Utils.Bn.G1.F.Zero(), Utils.Bn.G1.F.Zero()}
	proof.PiCp = [3]*big.Int{Utils.Bn.G1.F.Zero(), Utils.Bn.G1.F.Zero(), Utils.Bn.G1.F.Zero()}
	proof.PiH = [3]*big.Int{Utils.Bn.G1.F.Zero(), Utils.Bn.G1.F.Zero(), Utils.Bn.G1.F.Zero()}
	proof.PiKp = [3]*big.Int{Utils.Bn.G1.F.Zero(), Utils.Bn.G1.F.Zero(), Utils.Bn.G1.F.Zero()}

	for i := cir.NPublic + 1; i < cir.NVars; i++ {
		proof.PiA = Utils.Bn.G1.Add(proof.PiA, Utils.Bn.G1.MulScalar(setup.Pk.A[i], w[i]))
		proof.PiAp = Utils.Bn.G1.Add(proof.PiAp, Utils.Bn.G1.MulScalar(setup.Pk.Ap[i], w[i]))
	}

	for i := 0; i < cir.NVars; i++ {
		proof.PiB = Utils.Bn.G2.Add(proof.PiB, Utils.Bn.G2.MulScalar(setup.Pk.B[i], w[i]))
		proof.PiBp = Utils.Bn.G1.Add(proof.PiBp, Utils.Bn.G1.MulScalar(setup.Pk.Bp[i], w[i]))

		proof.PiC = Utils.Bn.G1.Add(proof.PiC, Utils.Bn.G1.MulScalar(setup.Pk.C[i], w[i]))
		proof.PiCp = Utils.Bn.G1.Add(proof.PiCp, Utils.Bn.G1.MulScalar(setup.Pk.Cp[i], w[i]))

		proof.PiKp = Utils.Bn.G1.Add(proof.PiKp, Utils.Bn.G1.MulScalar(setup.Pk.Kp[i], w[i]))
	}

	hx := Utils.PF.DivisorPolynomial(px, setup.Pk.Z)
	for i := 0; i < len(hx); i++ {
		proof.PiH = Utils.Bn.G1.Add(proof.PiH, Utils.Bn.G1.MulScalar(setup.G1T[i], hx[i]))
	}

	return proof, nil
}

// Verify verifies over the BN128 the Pairings of the Proof
func (setup *PinocchioSetup) Verify(proof Proof, publicSignals []*big.Int) (bool, error) {
	pproof, ok := proof.(*PinocchioProof)
	if !ok {
		return false, fmt.Errorf("bad proof type")
	}
	pairingPiaVa := Utils.Bn.Pairing(pproof.PiA, setup.Vk.Vka)
	pairingPiapG2 := Utils.Bn.Pairing(pproof.PiAp, Utils.Bn.G2.G)
	if !Utils.Bn.Fq12.Equal(pairingPiaVa, pairingPiapG2) {
		return false, nil
	}

	// e(Vb, piB) == e(piB', g2)
	pairingVbPib := Utils.Bn.Pairing(setup.Vk.Vkb, pproof.PiB)
	pairingPibpG2 := Utils.Bn.Pairing(pproof.PiBp, Utils.Bn.G2.G)
	if !Utils.Bn.Fq12.Equal(pairingVbPib, pairingPibpG2) {
		return false, nil
	}

	// e(piC, Vc) == e(piC', g2)
	pairingPicVc := Utils.Bn.Pairing(pproof.PiC, setup.Vk.Vkc)
	pairingPicpG2 := Utils.Bn.Pairing(pproof.PiCp, Utils.Bn.G2.G)
	if !Utils.Bn.Fq12.Equal(pairingPicVc, pairingPicpG2) {
		return false, nil
	}

	// Vkx+piA
	vkxpia := setup.Vk.IC[0]
	for i := 0; i < len(publicSignals); i++ {
		vkxpia = Utils.Bn.G1.Add(vkxpia, Utils.Bn.G1.MulScalar(setup.Vk.IC[i+1], publicSignals[i]))
	}

	// e(Vkx+piA, piB) == e(piH, Vkz) * e(piC, g2)
	if !Utils.Bn.Fq12.Equal(
		Utils.Bn.Pairing(Utils.Bn.G1.Add(vkxpia, pproof.PiA), pproof.PiB),
		Utils.Bn.Fq12.Mul(
			Utils.Bn.Pairing(pproof.PiH, setup.Vk.Vkz),
			Utils.Bn.Pairing(pproof.PiC, Utils.Bn.G2.G))) {
		return false, nil
	}

	// e(Vkx+piA+piC, g2KbetaKgamma) * e(g1KbetaKgamma, piB) == e(piK, g2Kgamma)
	piapic := Utils.Bn.G1.Add(Utils.Bn.G1.Add(vkxpia, pproof.PiA), pproof.PiC)
	pairingPiACG2Kbg := Utils.Bn.Pairing(piapic, setup.Vk.G2Kbg)
	pairingG1KbgPiB := Utils.Bn.Pairing(setup.Vk.G1Kbg, pproof.PiB)
	pairingL := Utils.Bn.Fq12.Mul(pairingPiACG2Kbg, pairingG1KbgPiB)
	pairingR := Utils.Bn.Pairing(pproof.PiKp, setup.Vk.G2Kg)
	if !Utils.Bn.Fq12.Equal(pairingL, pairingR) {
		return false, nil
	}

	return true, nil
}
