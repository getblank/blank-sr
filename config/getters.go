package config

import (
	"encoding/json"

	"github.com/ivahaev/go-logger"
)

func GetStoreObject(store string) (Model, bool) {
	mutex.RLock()
	defer mutex.RUnlock()
	o, ok := config[store]
	return o, ok
}

func GetAllStoreObjectsFromDb() map[string]Model {
	result := map[string]Model{}
	_o, err := DB.GetAll(bucket)
	if err != nil {
		return result
	}
	for _, v := range _o {
		var o Model
		err = json.Unmarshal(v, &o)
		if err != nil {
			return result
		}
		result[o.Store] = o
	}

	return result
}

func GetStoreObjectFromDb(store string) (Model, bool) {
	o := Model{}
	_o, err := DB.Get(bucket, store)
	if err != nil {
		return o, false
	}
	err = json.Unmarshal(_o, &o)
	if err != nil {
		return o, false
	}
	o.Store = store

	return o, true
}

func GetObjectsForUser(u User) map[string]*Model {
	stores, err := DB.GetAllKeys(bucket)
	if err != nil {
		logger.Error("Error when load stores keys")
		return nil
	}
	var hasDefault, hasDefaultProcess, hasDefaultCampaign, hasDefaultNotification, hasDefaultSingle bool
	var defaultDirectory, defaultProcess, defaultCampaign, defaultNotification, defaultSingle Model
	_default, ok := GetStoreObjectFromDb(DefaultDirectory)
	if ok {
		hasDefault = true
		defaultDirectory = _default
	}
	_defaultP, ok := GetStoreObjectFromDb(DefaultProcess)
	if ok {
		hasDefaultProcess = true
		defaultProcess = _defaultP
	}
	_defaultC, ok := GetStoreObjectFromDb(DefaultCampaign)
	if ok {
		hasDefaultCampaign = true
		defaultCampaign = _defaultC
	}
	_defaultN, ok := GetStoreObjectFromDb(DefaultNotification)
	if ok {
		hasDefaultNotification = true
		defaultNotification = _defaultN
	}
	_defaultS, ok := GetStoreObjectFromDb(DefaultSingle)
	if ok {
		hasDefaultSingle = true
		defaultSingle = _defaultS
	}
	result := map[string]*Model{}
	for _, store := range stores {
		switch store {
		case DefaultDirectory, DefaultProcess, DefaultCampaign, DefaultNotification:
			continue
		}
		v, ok := GetStoreObjectFromDb(store)
		if !ok {
			logger.Error("Error when load store", store)
			continue
		}
		if v.Type == ObjWorkspace {
			continue
		}
		if hasDefault {
			v.mergeFilters(&defaultDirectory)
			for _pName, _prop := range defaultDirectory.Props {
				v.LoadDefaultIntoProp(_pName, _prop)
			}
		}
		if hasDefaultProcess && v.Type == ObjProcess {
			v.mergeFilters(&defaultProcess)
			for _pName, _prop := range defaultProcess.Props {
				v.LoadDefaultIntoProp(_pName, _prop)
			}
		}
		if hasDefaultCampaign && v.Type == ObjCampaign {
			for _pName, _prop := range defaultCampaign.Props {
				v.LoadDefaultIntoProp(_pName, _prop)
			}
		}
		if hasDefaultNotification && v.Type == ObjNotification {
			v.mergeAccess(&defaultNotification)
			for _pName, _prop := range defaultNotification.Props {
				v.LoadDefaultIntoProp(_pName, _prop)
			}
		}
		if hasDefaultSingle && v.Type == ObjSingle {
			for _pName, _prop := range defaultSingle.Props {
				v.LoadDefaultIntoProp(_pName, _prop)
			}
		}
		v.preparePartialFlags()
		v.prepareI18nForUser(u)

		v.PrepareConfigForUser(u)
		v.prepareTemplate()
		if v.OwnerAccess == "" || v.OwnerAccess == "-" {
			continue
		}
		v.Access = nil
		result[store] = &v
	}
	if u.GetWorkspace() != "" {
		if workspace, ok := GetStoreObjectFromDb(u.GetWorkspace()); ok {
			for k, v := range workspace.Config {
				c, ok := result[k]
				if !ok {
					continue
				}
				v.prepareI18nForUser(u)
				mergeModels(&v, c)
				c.preparePartialFlags()
				c.prepareTemplate()
				result[k] = c
			}
		}
	}
	for _, v := range result {
		v.checkPropsRequiredConditions()
	}
	return result
}
