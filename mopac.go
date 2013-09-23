/*
 * qm.go, part of gochem.
 *
 *
 * Copyright 2012 Raul Mera <rmera{at}chemDOThelsinkiDOTfi>
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as
 * published by the Free Software Foundation; either version 2.1 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General
 * Public License along with this program.  If not, see
 * <http://www.gnu.org/licenses/>.
 *
 *
 * Gochem is developed at the laboratory for instruction in Swedish, Department of Chemistry,
 * University of Helsinki, Finland.
 *
 *
 */
/***Dedicated to the long life of the Ven. Khenpo Phuntzok Tenzin Rinpoche***/

package chem

import "os"
import "strings"
import "strconv"
import "bufio"
import "fmt"
import "os/exec"

const eV2Kcalmol float64 = 23.061

type MopacRunner struct {
	defmethod string
	command   string
	inputname string
}

//Creates and initialized a new instance of MopacRuner, with values set
//to its defaults.
func MakeMopacRunner() *MopacRunner {
	run := new(MopacRunner)
	run.SetDefaults()
	return run
}

//MopacRunner methods

//Just to satisfy the interface. It does nothing
func (O *MopacRunner) SetnCPU(cpu int) {
	//It does nothing! :-D
}

//Sets the name for the job, used for input
//and output files (ex. input will be name.inp).
func (O *MopacRunner) SetName(name string) {
	O.inputname = name
}

//Sets the command to run the MOPAC program.
func (O *MopacRunner) SetCommand(name string) {
	O.command = name
}

/*Sets some defaults for MopacRunner. default is an optimization at
  PM6-DH2X It tries to locate MOPAC2012 according to the
  $MOPAC_LICENSE environment variable, which might only work in UNIX.
  If other system or using MOPAC2009 the command Must be set with the
  SetCommand function. */
func (O *MopacRunner) SetDefaults() {
	O.defmethod = "PM6-D3H4"
	O.command = os.ExpandEnv("${MOPAC_LICENSE}/MOPAC2012.exe")
}

//BuildInput builds an input for ORCA based int the data in atoms, coords and C.
//returns only error.
func (O *MopacRunner) BuildInput(atoms ReadRef, coords *CoordMatrix, Q *QMCalc) error {
	if strings.Contains(Q.Others, "RI") {
		Q.Others = ""
	}
	//Only error so far
	if atoms == nil || coords == nil {
		return fmt.Errorf("Missing charges or coordinates")
	}
	ValidMethods := []string{"PM3", "PM6", "PM7", "AM1"}
	if !isInString(ValidMethods, Q.Method[0:3]) { //not found
		fmt.Fprintf(os.Stderr, "no method assigned for MOPAC calculation, will used the default %s, \n", O.defmethod)
		Q.Method = O.defmethod
	}
	opt := "" //Empty string means optimize
	if Q.Optimize == false {
		opt = "1SCF"
	}
	//If this flag is set we'll look for a suitable MO file.
	//If not found, we'll just use the default ORCA guess
	hfuhf := "RHF"
	if atoms.Unpaired() != 0 {
		hfuhf = "UHF"
	}
	cosmo := ""
	if Q.Dielectric > 0 {
		cosmo = fmt.Sprintf("EPS=%2.1f RSOLV=1.3 LET DDMIN=0.0", Q.Dielectric)  //The DDMIN ensures that the optimization continues when cosmo is used. From the manual I understand that it is OK
	}
	multi := mopacMultiplicity[atoms.Unpaired()+1]
	charge := fmt.Sprintf("CHARGE=%d", atoms.Charge())
	MainOptions := []string{hfuhf, Q.Method, opt, cosmo, charge, multi, Q.Others, "BONDS AUX\n"}
	mainline := strings.Join(MainOptions, " ")
	//Now lets write the thing
	if O.inputname == "" {
		O.inputname = "input"
	}
	file, err := os.Create(fmt.Sprintf("%s.mop", O.inputname))
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err = fmt.Fprint(file, "* ===============================\n* Input file for Mopac\n* ===============================\n"); err != nil {
		return err //After this check I just assume the file is ok and dont check again.
	}
	fmt.Fprint(file, mainline)
	fmt.Fprint(file, "\n")
	fmt.Fprint(file, "Mopac file generated by gochem :-)\n")
	//now the coordinates
	for i := 0; i < atoms.Len(); i++ {
		tag := 1
		if isInInt(Q.CConstraints, i) {
			tag = 0
		}
		//	fmt.Println(atoms.Atom(i).Symbol)
		fmt.Fprintf(file, "%-2s  %8.5f %d %8.5f %d %8.5f %d\n", atoms.Atom(i).Symbol, coords.At(i, 0), tag, coords.At(i, 1), tag, coords.At(i, 2), tag)
	}
	fmt.Fprintf(file, "\n")
	return nil
}

var mopacMultiplicity = map[int]string{
	1: "Singlet",
	2: "Doublet",
	3: "Triplet",
	4: "Quartet",
	5: "Quintet",
	6: "Sextet",
	7: "Heptet",
	8: "Octet",
	9: "Nonet",
}

//Run runs the command given by the string O.command
//it waits or not for the result depending on wait. Not waiting for results works
//only for unix-compatible systems, as it uses bash and nohup.
func (O *MopacRunner) Run(wait bool) (err error) {
	if wait == true {
		command := exec.Command(O.command, fmt.Sprintf("%s.mop", O.inputname))
		err = command.Run()
	} else {
		command := exec.Command("sh", "-c", "nohup "+O.command+fmt.Sprintf(" %s.mop &", O.inputname))
		err = command.Start()
	}
	return err
}

/*GetEnergy gets the last energy for a MOPAC2009/2012 calculation by
  parsing the mopac output file. Return error if fail. Also returns
  Error ("Probable problem in calculation")
  if there is a energy but the calculation didnt end properly*/
func (O *MopacRunner) GetEnergy() (float64, error) {
	var err error
	var energy float64
	file, err := os.Open(fmt.Sprintf("%s.out", O.inputname))
	if err != nil {
		return 0, err
	}
	defer file.Close()
	out := bufio.NewReader(file)
	err = fmt.Errorf("Mopac Energy not found in %s", O.inputname)
	trust_radius_warning := false
	for {
		var line string
		line, err = out.ReadString('\n')
		if err != nil {
			break
		}
		if strings.Contains(line, "TRUST RADIUS NOW LESS THAN 0.00010 OPTIMIZATION TERMINATING") {
			trust_radius_warning = true
			continue
		}
		if strings.Contains(line, "TOTAL ENERGY") {
			splitted := strings.Fields(line)
			if len(splitted) < 4 {
				err = fmt.Errorf("Error reading energy from MOPAC output file!")
				break
			}
			energy, err = strconv.ParseFloat(splitted[3], 64)
			if err != nil {
				break
			}
			energy = energy * eV2Kcalmol
			err = nil
			break
		}
	}
	if err != nil {
		return 0, err
	}
	if trust_radius_warning {
		err = fmt.Errorf("Probable problem in calculation")
	}
	return energy, err
}

/*Get Geometry reads the optimized geometry from a MOPAC2009/2012 output.
  Return error if fail. Returns Error ("Probable problem in calculation")
  if there is a geometry but the calculation didnt end properly*/
func (O *MopacRunner) GetGeometry(atoms Ref) (*CoordMatrix, error) {
	var err error
	natoms := atoms.Len()
	coords := make([]float64, natoms*3, natoms*3) //will be used for return
	file, err := os.Open(fmt.Sprintf("%s.out", O.inputname))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	out := bufio.NewReader(file)
	err = fmt.Errorf("Mopac Energy not found in %s", O.inputname)
	//some variables that will be changed/increased during the next for loop
	final_point := false //to see if we got to the right part of the file
	reading := false     //start reading
	i := 0
	errsl := make([]error, 3, 3)
	trust_radius_warning := false
	for {
		var line string
		line, err = out.ReadString('\n')
		if err != nil {
			break
		}

		if (!reading) && strings.Contains(line, "TRUST RADIUS NOW LESS THAN 0.00010 OPTIMIZATION TERMINATING") {
			trust_radius_warning = true
			continue
		}

		if !reading && (strings.Contains(line, "FINAL  POINT  AND  DERIVATIVES") || strings.Contains(line, "GEOMETRY OPTIMISED")) {
			final_point = true
			continue
		}
		if strings.Contains(line, "(ANGSTROMS)     (ANGSTROMS)     (ANGSTROMS)") && final_point {
			_, err = out.ReadString('\n')
			if err != nil {
				break
			}
			reading = true
			continue
		}
		if reading {
			//So far we dont check that there are not too many atoms in the mopac output.
			if i >= natoms {
				err = nil
				break
			}
			coords[i*3], errsl[0] = strconv.ParseFloat(strings.TrimSpace(line[22:35]), 64)
			coords[i*3+1], errsl[1] = strconv.ParseFloat(strings.TrimSpace(line[38:51]), 64)
			coords[i*3+2], errsl[2] = strconv.ParseFloat(strings.TrimSpace(line[54:67]), 64)
			i++
			err = parseErrorSlice(errsl)
			if err != nil {
				break
			}
		}
	}
	if err != nil {
		return nil, err
	}
	mcoords := NewCoords(coords)
	if trust_radius_warning {
		return mcoords, fmt.Errorf("Probable problem in calculation")
	}
	return mcoords, nil
}

//Support function, gets a slice of errors and returns the first
//non-nil error found, or nil if all errors are nil.
func parseErrorSlice(errorsl []error) error {
	for _, val := range errorsl {
		if val != nil {
			return val
		}
	}
	return nil
}
