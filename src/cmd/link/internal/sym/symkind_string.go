// Code generated by "stringer -type=SymKind"; DO NOT EDIT.

package sym

import "strconv"

const _SymKind_name = "SxxxSTEXTSELFRXSECTSTYPESSTRINGSGOSTRINGSGOFUNCSGCBITSSRODATASFUNCTABSELFROSECTSMACHOPLTSTYPERELROSSTRINGRELROSGOSTRINGRELROSGOFUNCRELROSGCBITSRELROSRODATARELROSFUNCTABRELROSTYPELINKSITABLINKSSYMTABSPCLNTABSELFSECTSMACHOSMACHOGOTSWINDOWSSELFGOTSNOPTRDATASINITARRSDATASBSSSNOPTRBSSSTLSBSSSXCOFFTOCSXREFSMACHOSYMSTRSMACHOSYMTABSMACHOINDIRECTPLTSMACHOINDIRECTGOTSFILEPATHSCONSTSDYNIMPORTSHOSTOBJSDWARFSECTSDWARFINFOSDWARFRANGESDWARFLOCSDWARFMISC"

var _SymKind_index = [...]uint16{0, 4, 9, 19, 24, 31, 40, 47, 54, 61, 69, 79, 88, 98, 110, 124, 136, 148, 160, 173, 182, 191, 198, 206, 214, 220, 229, 237, 244, 254, 262, 267, 271, 280, 287, 296, 301, 313, 325, 353, 359, 368, 374, 384, 392, 402, 412, 423, 432, 442}

func (i SymKind) String() string {
	if i >= SymKind(len(_SymKind_index)-1) {
		return "SymKind(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _SymKind_name[_SymKind_index[i]:_SymKind_index[i+1]]
}
