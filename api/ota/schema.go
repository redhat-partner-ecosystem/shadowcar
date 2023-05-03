package ota

type (
	Campaign struct {
		CampaignID      string         `json:"id,omitempty"`
		Name            string         `json:"name,omitempty"`
		Description     string         `json:"description,omitempty"`
		Priority        string         `json:"priority,omitempty"`
		Status          CampaignStatus `json:"status,omitempty"`
		LastModified    string         `json:"lastModified,omitempty"`
		VehicleGroupID  string         `json:"vehicle_group_id,omitempty"`
		UpdatePackeURI  string         `json:"update_package_uri,omitempty"`
		ReleaseNotesURI string         `json:"release_notes_uri,omitempty"`
	}

	Campaigns []Campaign

	CampaignStatus struct {
		Success       int `json:"success,omitempty"`
		Failure       int `json:"failure,omitempty"`
		TotalVehicles int `json:"total_vehicles,omitempty"`
		InProgress    int `json:"in_progress,omitempty"`
	}

	VehicleGroup struct {
		VehicleGroupID string   `json:"id,omitempty"`
		Name           string   `json:"name,omitempty"`
		VINS           []string `json:"vins,omitempty"`
	}
)
