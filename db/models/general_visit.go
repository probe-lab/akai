package models

type GeneralVisit struct {
	GenericVisit         []*SampleGenericVisit
	GenericValueVisit    []*SampleValueVisit
	GenericPeerInfoVisit []*PeerInfoVisit
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
