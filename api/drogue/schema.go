package drogue

type (
	Token struct {
		Prefix            string `json:"prefix,omitempty"` // The name of the application the resource is scoped to
		Description       string `json:"description"`
		CreationTimestamp string `json:"created,omitempty"`
	}

	Device struct {
		Metadata *ScopedMetadata `json:"metadata"`
		Spec     *DeviceSpec     `json:"spec,omitempty"`
		Status   *DeviceStatus   `json:"status,omitempty"`
	}

	ScopedMetadata struct {
		Application       string `json:"application,omitempty"` // The name of the application the resource is scoped to
		Name              string `json:"name"`
		UID               string `json:"uid,omitempty"`
		CreationTimestamp string `json:"creationTimestamp,omitempty"`
		DeletionTimestamp string `json:"deletionTimestamp,omitempty"`
		Generation        int    `json:"generation,omitempty"`
		ResourceVersion   string `json:"resourceVersion,omitempty"`
		// Finalizers
		// Annotations
		// Labels
	}

	DeviceSpec struct {
		Description     string                  `json:"description,omitempty"`
		Authentication  *DeviceCredentialStruct `json:"authentication,omitempty"`
		GatewaySelector *GatewaySelectorStruct  `json:"gatewaySelector,omitempty"`
		Alias           *AliasStruct            `json:"alias,omitempty"`
	}

	DeviceCredentialStruct struct {
		Description string      `json:"description,omitempty"`
		User        *UserStruct `json:"user,omitempty"`
		Pass        string      `json:"pass,omitempty"`
		//UserCredential UserCredentialStruct `json:"user"`
		//PassCredential PasswordStruct       `json:"pass"`
	}

	UserStruct struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	GatewaySelectorStruct struct {
		MatchName []string `json:"matchNames"`
	}

	AliasStruct struct {
		Aliases []string `json:"aliases"`
	}

	DeviceStatus struct {
		Conditions []ConditionStruct `json:"conditions"`
	}

	ConditionStruct struct {
		Type               string `json:"type"`
		Status             string `json:"status"`
		LastTransitionTime string `json:"lastTransitionTime"`
	}

	Devices []Device
	Tokens  []Token
)
