package config

import "github.com/getblank/blank-sr/bdb"

var (
	apiConfig = Store{
		Store:    ApiKeysBucket,
		Type:     ObjSingle,
		NavGroup: "config",
		NavOrder: 1000,
		Label:    "API Access Config",
		Access: []Access{
			{
				Role:        RootGuid,
				Permissions: "crud",
			},
		},
		Props: map[string]Prop{
			"keys": {
				Type:  PropObjectList,
				Label: "Keys and permissions",
				Props: map[string]Prop{
					"key": {
						Type:      PropString,
						Label:     "API Key",
						FormOrder: 10,
						Display:   "textInput",
						Required:  true,
					},
					"create": {
						Type:      PropBool,
						Label:     "Create",
						FormOrder: 30,
						Display:   "checkbox",
						Default:   true,
						Style:     bdb.M{"marginRight": "7px", "flex": "0 0 40px"},
					},
					"read": {
						Type:      PropBool,
						Label:     "Read",
						FormOrder: 40,
						Display:   "checkbox",
						Default:   true,
						Style:     bdb.M{"marginRight": "7px", "flex": "0 0 40px"},
					},
					"update": {
						Type:      PropBool,
						Label:     "Update",
						FormOrder: 50,
						Display:   "checkbox",
						Default:   true,
						Style:     bdb.M{"marginRight": "7px", "flex": "0 0 40px"},
					},
					"delete": {
						Type:      PropBool,
						Label:     "Delete",
						FormOrder: 60,
						Display:   "checkbox",
						Default:   true,
						Style:     bdb.M{"marginRight": "7px", "flex": "0 0 40px"},
					},
				},
			},
		},
	}

	apiPropsStoreProp = Prop{
		Type:      PropString,
		Label:     "Store",
		FormOrder: 20,
		Display:   "select",
		Options:   apiPropsStoreOptions,
		Required:  true,
		Style:     bdb.M{"marginRight": "7px", "flex": "0 0 160px"},
	}

	apiPropsStoreOptions = []interface{}{}
)
