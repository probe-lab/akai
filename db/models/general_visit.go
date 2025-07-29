package models

type GeneralVisit struct {
	GenericVisit           []*SampleGenericVisit
	GenericValueVisit      []*SampleValueVisit
	GenericPeerInfoVisit   []*PeerInfoVisit
	GenericIPNSRecordVisit []*IPNSRecordVisit
}

func (v *GeneralVisit) GetGenericVisit() []*SampleGenericVisit {
	if v != nil {
		return v.GenericVisit
	}
	return nil
}

func (v *GeneralVisit) GetGenericValueVisit() []*SampleValueVisit {
	if v != nil {
		return v.GenericValueVisit
	}
	return nil
}

func (v *GeneralVisit) GetGenericPeerInfoVisit() []*PeerInfoVisit {
	if v != nil {
		return v.GenericPeerInfoVisit
	}
	return nil
}

func (v *GeneralVisit) GetGenericIPNSRecordVisit() []*IPNSRecordVisit {
	if v != nil {
		return v.GenericIPNSRecordVisit
	}
	return nil
}
