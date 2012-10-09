
// +build !gromacs

/*
 * general_test.go
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
 */

/*This provides some tests for the library, in the form of little functions 
 * that have practical applications*/

package chem


import "github.com/skelterjohn/go.matrix"
import "fmt"
import "testing"
import "os"




//TestChangeAxis reads the PDB 2c9v.pdb from the test directory, collects 
//The CA and CB of residue D124 of the chain A, and rotates the 
//whole molecule such as the vector defined by these 2 atoms is 
//aligned with the Z axis. The new molecule is written
//as 2c9v_aligned.pdb to the test folder.
func TesaChangeAxis(Te *testing.T){
	mol,err:=PdbRead("test/2c9v.pdb",true)
	if err!=nil{
		Te.Error(err)
		}
	orient_atoms:=[2]int{0,0}
	for index:=0;index<mol.Len();index++{
		atom:=mol.Atom(index)
		if atom.Chain=='A' && atom.Molid==124{
			if atom.Name=="CA"{
				orient_atoms[0]=index
				}else if atom.Name=="CB"{
				orient_atoms[1]=index	
				}
			}
		}
	ov1:=mol.Coord(orient_atoms[0], 0)
	ov2:=mol.Coord(orient_atoms[1], 0)
	//now we center the thing in the beta carbon of D124
	err=SubRow(mol.Coords[0],ov2)
	//Now the rotation
	ov1=mol.Coord(orient_atoms[0], 0) //make sure we have the correct versions
	ov2=mol.Coord(orient_atoms[1], 0)  //same
	orient:=ov2.Copy()	
	orient.SubtractDense(ov1)
	rotation:=GetSwitchZ(orient)
	fmt.Println("rotation: ",rotation)
	mol.Coords[0]=matrix.ParallelProduct(mol.Coords[0],rotation)
	fmt.Println(orient_atoms[1], mol.Atom(orient_atoms[1]),mol.Atom(orient_atoms[0]))
	if err!=nil{
		Te.Error(err)
		}
	PdbWrite(mol,"test/2c9v-aligned.pdb")
	}


//TestGeo opens the sample.xyz file in the test directory, and pull a number of hardcoded atoms
//In the direction of a hardcoded vectos. It builds 12 files with the pulled atoms  displaced by
//different ammounts along the pulling vector
func TestGeo(Te *testing.T) {
	pulled_atoms:=[7]int{43,41,42,40,85,86,87}
	pulling_vector:=[2]int{40,88}
	mol,err:=XyzRead("test/sample.xyz")
	if err!=nil{
		Te.Error(err)
		}
	pulled_res:=SomeRows(mol.Coords[0],pulled_atoms[:])
	at1:=mol.Coord(pulling_vector[0],0)
	vector:=mol.Coord(pulling_vector[1],0)
	vector=vector.Copy()
	err=vector.SubtractDense(at1)
	if err!=nil{
		Te.Error(err)
		}
	vector=Unitarize(vector)
	var scale_factors = [12]float64{-1.0, -2.0, 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}
	for _,scaling:=range(scale_factors){
		vec:=vector.Copy()
		pulled:=pulled_res.Copy()
		vec.Scale(scaling)
		err=AddRow(pulled,vec)
		if err!=nil{
			Te.Error(err)
			}
		mol.SetCoords(pulled_atoms[:], 0, pulled)
		err=mol.Corrupted()
		if err!=nil{
			Te.Error(err)
			}
		XyzWrite(mol, 0, fmt.Sprintf("test/sample_%03.1f.xyz",scaling))
		}
	}

func TestRama(Te *testing.T){
	mol,err:=PdbRead("test/2c9v.pdb",true)
	if err!=nil{
		Te.Error(err)
		}
	ramalist, err:=RamaList(mol,"A",[]int{0,-1}) ////
	if err!=nil{
		Te.Error(err)
		}
	rama,err := RamaCalc(mol.Coords[0],ramalist)
	if err!=nil{
		Te.Error(err)
		}
	fmt.Println("Rama",rama,len(rama),len(ramalist),mol.Len())
	err=RamaPlot(rama,"test/Rama")
	if err!=nil{
		Te.Error(err)
		}
	PdbWrite(mol,"test/Used4Rama.pdb")
	//for the 3 residue  I should get -131.99, 152.49.
	}

func TestQM(Te *testing.T) {
	mol,err:=XyzRead("test/sample.xyz")
	if err!=nil{
		Te.Error(err)
		}
	if err:=mol.Corrupted();err!=nil{
		Te.Error(err)
		}
	mol.Del(mol.Len()-1)
	mol.SetCharge(1)
	mol.SetUnpaired(0)
	calc:=new(QMCalc)
	calc.SCFTightness=3 //very demanding
	calc.Optimize=true
	calc.Method="BLYP"
	calc.Dielectric=4
	calc.Basis="def2-TZVPP"
	calc.HighBasis="def2-QZVPP"
	calc.HBAtoms=[]int{3,10,12}
	calc.HBElements=[]string{"Cu","Zn"}
	calc.AuxBasis="def2-TZVPP/J"
	calc.Disperssion="D2"
	calc.CConstraints=[]int{0,10,20}
	orca:=MakeOrcaRunner()
	atoms,_:=mol.Next(true)
	original_dir,_:=os.Getwd() //will check in a few lines
	if err=os.Chdir("./test"); err!=nil{
		Te.Error(err)
		}
	_=orca.BuildInput(mol,atoms,calc)
	path,_:=os.Getwd()
	fmt.Println(path)
//Took away this because it takes too long to run :-)
/*	if err=orca.Run(true); err!=nil{
		Te.Error(err)
		}
*/
	if err=os.Chdir(original_dir);err!=nil{
		Te.Error(err)
		}
	}


func TestMatrix(Te *testing.T){
	a:=[]float64{1,1,4,2,2,5,3,3,6}
	A:=matrix.MakeDenseMatrix(a,3,3)
	fmt.Println("before:\n",A)
	A,err:=DMDelRow(A,1)
	if err!=nil{
		Te.Error(err)
		}
	fmt.Println("after:\n",A)
	}

