package entity

// The admin request is a superset of write request features
func ToAdminWriteEntityRequest(req *WriteEntityRequest) *AdminWriteEntityRequest {
	return &AdminWriteEntityRequest{
		Entity: &Entity{
			GRN:     req.Entity.GRN,
			Meta:    req.Entity.Meta,
			Body:    req.Entity.Body,
			Status:  req.Entity.Status,
			Folder:  req.Entity.Folder,
			Message: req.Entity.Message,
		},
		PreviousVersion: req.PreviousVersion,
	}
}
