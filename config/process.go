package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/getblank/blank-sr/bdb"
	"github.com/getblank/blank-sr/utils/array"

	"github.com/imdario/mergo"
	"github.com/ivahaev/go-logger"
)

var (
	conf           map[string]Model
	mustacheRgx    = regexp.MustCompile(`(?U)({{.+}})`)
	handleBarseRgx = regexp.MustCompile(`{?{{\s*(\w*)\s?(\w*)?\s?.*}}`)
	itemPropsRgx   = regexp.MustCompile(`\$item.([A-Za-z][A-Za-z0-9]*)`)
	actionIdRgx    = regexp.MustCompile(`^[A-Za-z_]+[A-Za-z0-9_]*$`)
)

func Init(confFile string) {
	makeDefaultSettings()
	mustReadConfig(confFile)
}

func mustReadConfig(confFile string) {
	logger.Info("Try to load config from: " + confFile)
	file, err := ioutil.ReadFile(confFile)
	if err != nil {
		if confFile == "config.json" {
			logger.Notice("Can't find 'config.json'. Will work with saved config.")
			time.Sleep(time.Microsecond * 200)
			return
		} else {
			logger.Error(fmt.Sprintf("Config file read error: %v", err.Error()))
			time.Sleep(time.Microsecond * 200)
			os.Exit(1)
		}
	}

	err = json.Unmarshal(file, &conf)
	if err != nil {
		logger.Error("Error when read objects config", err.Error())
		time.Sleep(time.Microsecond * 200)
		os.Exit(1)
	}
	loadCommonSettings()
	loadServerSettings()
	validateConfig()
}

func loadCommonSettings() {
	cs, ok := conf[ObjCommonSettings]
	if !ok {
		logger.Warn("No common settings in config")
		return
	}
	encoded, err := json.Marshal(cs.Entries)
	if err != nil {
		logger.Error("Can't marshal common settings", cs.Entries, err.Error())
	} else {
		err = json.Unmarshal(encoded, CommonSettings)
		if err != nil {
			logger.Error("Can't unmarshal common settings", string(encoded), err.Error())
		}
	}
	encoded, err = json.Marshal(cs.I18n)
	if err != nil {
		logger.Error("Can't marshal common i18n", cs.I18n, err.Error())
		return
	}
	err = json.Unmarshal(encoded, &CommonSettings.I18n)
	if err != nil {
		logger.Error("Can't unmarshal common i18n", string(encoded), err.Error())
	}
}

func loadServerSettings() {
	ss, ok := conf[ObjServerSettings]
	if !ok {
		logger.Warn("No server settings in config")
		return
	}
	encoded, err := json.Marshal(ss.Entries)
	if err != nil {
		logger.Error("Can't marshal server settings", ss.Entries, err.Error())
		return
	}
	err = json.Unmarshal(encoded, ServerSettings)
	if err != nil {
		logger.Error("Can't unmarshal server settings", string(encoded), err.Error())
	}
}

func validateConfig() {
	mutex.Lock()
	defer mutex.Unlock()
	_conf := map[string]Model{}
	var err error

	for store, o := range conf {
		logger.Info("Parsing config for store:", store)
		o.Store = store
		if o.Props == nil {
			o.Props = map[string]Prop{}
		}
		if o.HeaderProperty == "" {
			o.HeaderProperty = "name"
		}

		// Checking object type
		switch o.Type {
		case ObjDirectory:
			//			logger.Info("Store is 'directory' type")
		case ObjProcess:
			//			logger.Info("Store is 'process' type")
		case ObjMap:
			//			logger.Info("Store is 'inConfigSet' type")
			o.Props = nil
		case ObjWorkspace:
			//			logger.Info("Store is 'workspace' type")
			o.Props = nil
		case ObjCampaign:
			//			logger.Info("Store is 'campaign' type")
		case ObjNotification:
			//			logger.Info("Store is 'notification' type")
		case ObjSingle:
			//			logger.Info("Store is 'single' type")
		case ObjFile:
			// 		logger.Info("Store is 'file' type")
		case ObjProxy:
			// 		logger.Info("Store is 'proxy' type")
		default:
			o.Type = ObjDirectory
		}

		allPropsValid := true

		err = o.validateProps(o.Props, true)
		if err != nil {
			logger.Error("Validating props failed:", err)
			allPropsValid = false
			continue
		}

		// prepare HtmlFile for props
		if err = o.preparePropHtmlTemplates(); err != nil {
			logger.Error("Preparing HTML templates failed:", err)
			allPropsValid = false
			continue
		}

		//compile actions
		if err = o.compileActions(); err != nil {
			logger.Error("Compiling actions failed:", err)
			allPropsValid = false
			continue
		}

		//compile hooks
		if err = o.prepareHooks(true); err != nil {
			logger.Error("Preparing hooks failed:", err)
			allPropsValid = false
			continue
		}

		//create tasks
		if err = o.createTasks(); err != nil {
			logger.Error("Creating tasks failed:", err)
			allPropsValid = false
			continue
		}

		if allPropsValid {
			_conf[store] = o
		} else {
			logger.Error("Invalid Store", store, o)
		}

		o.checkPropsRequiredConditions()

		// checking for httpApi enabled
		if o.HttpApi {
			apiPropsStoreOptions = append(apiPropsStoreOptions, bdb.M{"label": o.Store, "value": o.Store})
		}
	}
	if len(apiPropsStoreOptions) > 0 {
		apiPropsStoreProp.Options = apiPropsStoreOptions
		apiConfig.Props["keys"].Props["store"] = apiPropsStoreProp
		_conf[ApiKeysBucket] = apiConfig
	}

	// Place to save conf in DB
	DB.DeleteBucket(bucket)
ConfLoop:
	for storeName := range _conf {
		store := _conf[storeName]
		for name, p := range store.Props {
			if p.Type == PropRef || p.Type == PropRefList || p.Type == PropVirtualRefList {
				_, ok := _conf[p.Store]
				if !ok {
					logger.Error("Oppostite store '" + p.Store + "' not exists for prop '" + name + "' in store '" + storeName + "'. Store will ignored!")
					continue ConfLoop
				}
			}

			for subName, subP := range p.Props {
				if subP.Type == PropRef || subP.Type == PropRefList || subP.Type == PropVirtualRefList {
					_, ok := _conf[subP.Store]
					if !ok {
						logger.Error("Oppostite store '" + subP.Store + "' not exists for prop '" + name + "." + subName + "' in store '" + storeName + "'. Store will ignored!")
						continue ConfLoop
					}
				}
			}
		}

		switch storeName {
		case DefaultDirectory, DefaultSingle, DefaultCampaign, DefaultNotification, DefaultProcess:
			//			logger.Info("This is", store, "store")
		default:
			if defaultDirectory, ok := _conf[DefaultDirectory]; ok {
				for _pName, _prop := range defaultDirectory.Props {
					store.LoadDefaultIntoProp(_pName, _prop)
				}
			}
			switch store.Type {
			case ObjProcess:
				if defaultProcess, ok := _conf[DefaultProcess]; ok {
					for _pName, _prop := range defaultProcess.Props {
						store.LoadDefaultIntoProp(_pName, _prop)
					}
				}
			case ObjCampaign:
				if defaultCampaign, ok := _conf[DefaultCampaign]; ok {
					for _pName, _prop := range defaultCampaign.Props {
						store.LoadDefaultIntoProp(_pName, _prop)
					}
				}
			case ObjNotification:
				if defaultNotification, ok := _conf[DefaultNotification]; ok {
					for _pName, _prop := range defaultNotification.Props {
						store.LoadDefaultIntoProp(_pName, _prop)
					}
				}
			case ObjSingle:
				if defaultSingle, ok := _conf[DefaultSingle]; ok {
					for _pName, _prop := range defaultSingle.Props {
						store.LoadDefaultIntoProp(_pName, _prop)
					}
				}
			}
			// if len(store.Access) == 0 {
			// 	store.Access = []Access{
			// 		{
			// 			Role:        "all",
			// 			Permissions: "crud",
			// 		},
			// 	}
			// }
			err := DB.Save(bucket, storeName, store)
			if err != nil {
				logger.Error("Error when saving store in conf", err.Error())
			}
		}

		config[store.Store] = store
		if store.HttpApi {
			HttpApiEnabledStores = append(HttpApiEnabledStores, store)
		}
	}

	for storeName, _store := range config {
		if _store.Type == ObjProxy {

			baseStore, ok := config[_store.BaseStore]
			if !ok {
				logger.Error("Can't find baseStore " + _store.BaseStore + " for proxy store " + _store.Store)
				delete(config, _store.Store)
				continue
			}
			if baseStore.Proxies == nil {
				baseStore.Proxies = []string{}
			}
			baseStore.Proxies = append(baseStore.Proxies, _store.Store)
			config[baseStore.Store] = baseStore

			// cloning base store
			encoded, _ := json.Marshal(baseStore)
			var store Model
			json.Unmarshal(encoded, &store)

			store.Store = storeName
			store.BaseStore = _store.BaseStore
			store.Type = ObjProxy

			if _store.Access != nil {
				store.Access = _store.Access
			}
			store.Actions = _store.Actions
			if _store.NavOrder != 0 {
				store.NavOrder = _store.NavOrder
			}
			if _store.NavGroup != "" {
				store.NavGroup = _store.NavGroup
			}
			if _store.Display != "" {
				store.Display = _store.Display
			}
			if _store.HeaderTemplate != "" {
				store.HeaderTemplate = _store.HeaderTemplate
			}
			if _store.HeaderProperty != "" {
				store.HeaderProperty = _store.HeaderProperty
			}
			if _store.Filters != nil {
				store.Filters = _store.Filters
			}
			if _store.Labels != nil {
				store.Labels = _store.Labels
			}

			err := DB.Save(bucket, store.Store, store)
			if err != nil {
				logger.Error("Error when saving object in conf", err.Error())
			}
			_store = store
		}
	}
}

func (m *Model) preparePropHtmlTemplates() (err error) {
	// Temporarly disabling loading html files

	// for i := range m.Props {
	// 	prop := m.Props[i]
	// 	if m.Props[i].HtmlFile != "" {
	// 		bytes, err := ioutil.ReadFile("local/lib/" + prop.HtmlFile)
	// 		if err != nil {
	// 			bytes, err = ioutil.ReadFile(prop.HtmlFile)
	// 		}
	// 		if err != nil {
	// 			logger.Error("Can't read HtmlFile " + prop.HtmlFile + " for prop " + i)
	// 			return err
	// 		}
	// 		prop.Html = string(bytes)
	// 	}
	// 	for j := range prop.Props {
	// 		subProp := prop.Props[j]
	// 		if subProp.HtmlFile != "" {
	// 			bytes, err := ioutil.ReadFile("local/lib/" + subProp.HtmlFile)
	// 			if err != nil {
	// 				bytes, err = ioutil.ReadFile(subProp.HtmlFile)
	// 			}
	// 			if err != nil {
	// 				logger.Error("Can't read HtmlFile " + subProp.HtmlFile + " for prop " + i + "." + j)
	// 				return err
	// 			}
	// 			subProp.Html = string(bytes)
	// 			prop.Props[j] = subProp
	// 		}
	// 	}
	// 	m.Props[i] = prop
	// }
	return nil
}

func (m *Model) compileActions() (err error) {
	var actionIds = []string{}
	if m.Actions != nil && len(m.Actions) > 0 {
		for i, a := range m.Actions {
			if !actionIdRgx.MatchString(a.Id) {
				return errors.New("Invalid action name. Must start with a letter or underscore and contains only letters, digits or underscores")
			}
			if a.Type == "client" {
				continue
			}
			actionIds = append(actionIds, a.Id)
			if a.Script != "" {
				script := a.Script
				if a.Disabled != nil {
					switch a.Disabled.(type) {
					case string:
						disabled := a.Disabled.(string)
						script = `if (` + disabled + `) {console.error("Action is disabled"); return "Action is disabled"};
						` + script
					case bool:
						disabled := a.Disabled.(bool)
						if disabled {
							script = `console.error("Action is disabled"); return "Action is disabled"`
						}
					default:
						return errors.New("Invalid action " + a.Id + ". Invalid Disabled property")
					}
				}
				if a.Hidden != nil {
					switch a.Hidden.(type) {
					case string:
						hidden := a.Hidden.(string)
						script = `if (` + hidden + `) {console.error("Action is hidden"); return "Action is hidden"};
						` + script
					case bool:
						hidden := a.Hidden.(bool)
						if hidden {
							script = `console.error("Action is hidden"); return "Action is hidden"`
						}
					default:
						return errors.New("Invalid action " + a.Id + ". Invalid hidden property")
					}
				}
				scriptId := m.Store + "_action_" + a.Id
				m.Actions[i].ScriptId = scriptId
				if a.Type == "http" {
					Scripts[scriptId] = `function($user, $data, $item, $request, $response){` + script + `}`
				} else {
					Scripts[scriptId] = `function($user, $data, $item){` + script + `}`
				}
			}
			for k, v := range m.Actions[i].Props {
				if v.Type == "" {
					v.Type = PropString
				}
				m.Actions[i].Props[k] = v
			}
			if a.ConcurentCallsLimit > 0 {
				id := m.Store + "actions" + a.Id
				concurrentChannels[id] = make(chan struct{}, a.ConcurentCallsLimit)
			}
		}
	}
	sort.Strings(actionIds)
	if m.StoreActions != nil && len(m.StoreActions) > 0 {
		for i, a := range m.StoreActions {
			if !actionIdRgx.MatchString(a.Id) {
				return errors.New("Invalid action name. Must start with a letter or underscore and contains only letters, digits or underscores")
			}
			if len(actionIds) > 0 && array.IndexOfSortedStrings(actionIds, a.Id) != -1 {
				return errors.New("Can't create store action with _id " + a.Id + " for store " + m.Store + " because action is present with the same _id")
			}
			if a.Script != "" {
				script := a.Script
				if a.Disabled != nil {
					switch a.Disabled.(type) {
					case string:
						disabled := a.Disabled.(string)
						script = `if (` + disabled + `) {console.error("Action is disabled"); return "Action is disabled"};
						` + script
					case bool:
						disabled := a.Disabled.(bool)
						if disabled {
							script = `console.error("Action is disabled"); return "Action is disabled"`
						}
					default:
						return errors.New("Invalid action " + a.Id + ". Invalid disabled property")
					}
				}
				if a.Hidden != nil {
					switch a.Hidden.(type) {
					case string:
						hidden := a.Hidden.(string)
						script = `if (` + hidden + `) {console.error("Action is hidden"); return "Action is hidden"};
						` + script
					case bool:
						hidden := a.Hidden.(bool)
						if hidden {
							script = `console.error("Action is hidden"); return "Action is hidden"`
						}
					default:
						return errors.New("Invalid action " + a.Id + ". Invalid hidden property")
					}
				}
				scriptId := m.Store + "_storeAction_" + a.Id
				m.StoreActions[i].ScriptId = scriptId
				if a.Type == "http" {
					Scripts[scriptId] = `function($user, $data, $filter, $request, $response){` + script + `}`
				} else {
					Scripts[scriptId] = `function($user, $data, $filter){` + script + `}`
				}
				if a.ConcurentCallsLimit > 0 {
					id := m.Store + "actions" + a.Id
					concurrentChannels[id] = make(chan struct{}, a.ConcurentCallsLimit)
				}
			}
		}
	}
	return nil
}

func (m *Model) prepareHooks(compile bool) (err error) {
	if m.ObjectLifeCycle.WillCreate != "" {
		scriptId := m.Store + "_willCreate"
		m.ObjectLifeCycle.WillCreateScriptId = scriptId
		if compile {
			Scripts[scriptId] = `function($user, $item, $lastComment){` + m.ObjectLifeCycle.WillCreate + `}`
			script := "(function(){" + m.ObjectLifeCycle.WillCreate + "}())"
			script = `var error = ` + script + `; if (error) {return {"error": error, "$item": $item}}; return {"$item": $item}`
			Scripts[scriptId+"Go"] = `function($user, $item, $lastComment){` + script + `}`

		}
	}
	if m.ObjectLifeCycle.DidCreate != "" {
		scriptId := m.Store + "_didCreate"
		m.ObjectLifeCycle.DidCreateScriptId = scriptId
		if compile {
			script := "(function(){" + m.ObjectLifeCycle.DidCreate + "}())"
			script = script
			Scripts[scriptId] = `function($user, $item, $lastComment){` + script + `}`
			Scripts[scriptId+"Go"] = `function($user, $item, $lastComment){` + script + `}`
		}
	}
	if m.ObjectLifeCycle.WillSave != "" {
		scriptId := m.Store + "_willSave"
		m.ObjectLifeCycle.WillSaveScriptId = scriptId
		if compile {
			Scripts[scriptId] = `function($user, $item, $lastComment, $prevItem){` + m.ObjectLifeCycle.WillSave + `}`
			script := "(function(){" + m.ObjectLifeCycle.WillSave + "}())"
			script = `var error = ` + script + `; if (error) {return {"error": error, "$item": $item}}; return {"$item": $item}`
			Scripts[scriptId+"Go"] = `function($user, $item, $lastComment, $prevItem){` + script + `}`

		}
	}
	if m.ObjectLifeCycle.DidSave != "" {
		scriptId := m.Store + "_didSave"
		m.ObjectLifeCycle.DidSaveScriptId = scriptId
		if compile {
			script := "(function(){" + m.ObjectLifeCycle.DidSave + "}())"
			script = script
			Scripts[scriptId] = `function($user, $item, $lastComment, $prevItem){` + script + `}`
			Scripts[scriptId+"Go"] = `function($user, $item, $lastComment, $prevItem){` + script + `}`
		}
	}
	if m.ObjectLifeCycle.WillRemove != "" {
		scriptId := m.Store + "_willRemove"
		m.ObjectLifeCycle.WillRemoveScriptId = scriptId
		if compile {
			script := "(function(){" + m.ObjectLifeCycle.WillRemove + "}())"
			script = `var error = ` + script + `; if (error) {return {"error": error, "$item": $item}}; return {"$item": $item}`
			Scripts[scriptId] = `function($user, $item, $lastComment){` + script + `}`
			Scripts[scriptId+"Go"] = `function($user, $item, $lastComment){` + script + `}`
		}
	}
	if m.ObjectLifeCycle.DidRemove != "" {
		scriptId := m.Store + "_didRemove"
		m.ObjectLifeCycle.DidRemoveScriptId = scriptId
		if compile {
			Scripts[scriptId] = `function($user, $item, $lastComment){` + m.ObjectLifeCycle.DidRemove + `}`
			Scripts[scriptId+"Go"] = `function($user, $item, $lastComment){` + m.ObjectLifeCycle.DidRemove + `}`
		}
	}
	if m.ObjectLifeCycle.DidRead != "" {
		scriptId := m.Store + "_didRead"
		m.ObjectLifeCycle.DidReadScriptId = scriptId
		if compile {
			Scripts[scriptId] = `function($user, $item, $lastComment){` + m.ObjectLifeCycle.DidRead + `}`
			Scripts[scriptId+"Go"] = `function($user, $item, $lastComment){` + m.ObjectLifeCycle.DidRead + `}`
		}
	}

	if m.StoreLifeCycle.WillCreate != "" {
		scriptId := m.Store + "_storeWillCreate"
		m.StoreLifeCycle.WillCreateScriptId = scriptId
		if compile {
			Scripts[scriptId] = `function($item){` + m.StoreLifeCycle.WillCreate + `}`
			Scripts[scriptId+"Go"] = `function($item){` + m.StoreLifeCycle.WillCreate + `}`
		}
	}
	if m.StoreLifeCycle.DidCreate != "" {
		scriptId := m.Store + "_storeDidCreate"
		m.StoreLifeCycle.DidCreateScriptId = scriptId
		if compile {
			Scripts[scriptId] = `function($item){` + m.StoreLifeCycle.DidCreate + `}`
			Scripts[scriptId+"Go"] = `function($item){` + m.StoreLifeCycle.DidCreate + `}`
		}
	}
	if m.StoreLifeCycle.WillSave != "" {
		scriptId := m.Store + "_storeWillSave"
		m.StoreLifeCycle.WillSaveScriptId = scriptId
		if compile {
			Scripts[scriptId] = `function($item){` + m.StoreLifeCycle.WillCreate + `}`
			Scripts[scriptId+"Go"] = `function($item){` + m.StoreLifeCycle.WillCreate + `}`
		}
	}
	if m.StoreLifeCycle.DidSave != "" {
		scriptId := m.Store + "_storeDidSave"
		m.StoreLifeCycle.DidCreateScriptId = scriptId
		if compile {
			Scripts[scriptId] = `function($item){` + m.StoreLifeCycle.DidSave + `}`
			Scripts[scriptId+"Go"] = `function($item){` + m.StoreLifeCycle.DidSave + `}`
		}
	}
	if m.StoreLifeCycle.WillRemove != "" {
		scriptId := m.Store + "_storeWillRemove"
		m.StoreLifeCycle.WillRemoveScriptId = scriptId
		if compile {
			Scripts[scriptId] = `function($item){` + m.StoreLifeCycle.WillRemove + `}`
			Scripts[scriptId+"Go"] = `function($item){` + m.StoreLifeCycle.WillRemove + `}`
		}
	}
	if m.StoreLifeCycle.DidRemove != "" {
		scriptId := m.Store + "_storeDidRemove"
		m.StoreLifeCycle.DidRemoveScriptId = scriptId
		if compile {
			Scripts[scriptId] = `function($item){` + m.StoreLifeCycle.DidRemove + `}`
			Scripts[scriptId+"Go"] = `function($item){` + m.StoreLifeCycle.DidRemove + `}`
		}
	}
	if m.StoreLifeCycle.DidStart != "" {
		scriptId := m.Store + "_storeDidStart"
		m.StoreLifeCycle.DidStartScriptId = scriptId
		if compile {
			Scripts[scriptId] = `function($item){` + m.StoreLifeCycle.DidStart + `}`
			Scripts[scriptId+"Go"] = `function($item){` + m.StoreLifeCycle.DidStart + `}`
		}
	}
	for k, v := range m.HttpHooks {
		switch v.Method {
		case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
			//				logger.Info("This is httpHook with", v.Method, "method")
		default:
			logger.Error("Unknown http method in httpHook", v.Method)
			return errors.New("Unknown http method in httpHook")
		}
		scriptId := m.Store + "_httpHook_" + v.Method + "_" + v.Uri
		m.HttpHooks[k].ScriptId = scriptId
		if compile {
			Scripts[scriptId] = `function($request, $response){` + v.Script + `}`
			if v.ConcurentCallsLimit > 0 {
				id := m.Store + "httpHook" + v.Uri
				concurrentChannels[id] = make(chan struct{}, v.ConcurentCallsLimit)
			}
		}
	}
	if compile {
		UpdateChannel <- *m
	}

	return nil
}

func (m *Model) createTasks() error {
	for i, t := range m.Tasks {
		t.ScriptId = m.Store + "_task_" + strconv.Itoa(i)
		Scripts[t.ScriptId] = `function(){` + t.Script + `}`
	}
	return nil
}

func (m *Model) validateProps(props map[string]Prop, parseObjects bool) error {
	var virtualPropsLoaders = map[string]string{}
	for pName, prop := range props {
		prop.Name = pName
		// Processing Type
		if prop.Type == "" {
			prop.Type = PropString
		}
		switch prop.Type {
		case PropWidget, PropAction, PropFile, PropFileList:
			continue
		case PropDynamic:
			continue
		case PropInt:
			prop.clearStringParams()
			prop.clearRefParams()
			prop.clearObjectParams()
			if prop.Default != nil {
				_, ok := prop.checkDefaultInt()
				if !ok {
					return errors.New("Wrong default int value in prop: '" + pName + "'")
				}
			}
			_, _, ok := prop.checkMinMaxParams()
			if !ok {
				return errors.New("Wrong min-max params in prop: '" + pName + "'")
			}
			//			if prop.Values != nil && len(prop.Values) > 0 {
			//				for _, v := range prop.Values {
			//					if _, ok := v.Value.(float64); !ok {
			//						return errors.New("Invalid int value in list in prop: '" + pName + "'")
			//					}
			//				}
			//			}
			props[pName] = prop
		case PropFloat:
			prop.clearStringParams()
			prop.clearRefParams()
			prop.clearObjectParams()
			if prop.Default != nil {
				_, ok := prop.checkDefaultFloat()
				if !ok {
					return errors.New("Wrong default float value in prop: '" + pName + "'")
				}
			}
			_, _, ok := prop.checkMinMaxParams()
			if !ok {
				return errors.New("Wrong min-max params in prop: '" + pName + "'")
			}
			props[pName] = prop
		case PropBool:
			prop.clearStringParams()
			prop.clearRefParams()
			prop.clearNumberParams()
			prop.clearObjectParams()
			if prop.Default != nil {
				_, ok := prop.Default.(bool)
				if !ok {
					return errors.New("Wrong default bool value in prop: '" + pName + "'")
				}
			}
			props[pName] = prop
		case PropString:
			prop.clearNumberParams()
			prop.clearRefParams()
			prop.clearObjectParams()
			if prop.Default != nil {
				_, ok := prop.Default.(string)
				if !ok {
					return errors.New("Wrong default string value in prop: '" + pName + "'")
				}
			}
			if prop.MinLength < 0 || prop.MaxLength < 0 {
				return errors.New("Wrong minLength or maxLength values in prop: '" + pName + "'")
			}
			if prop.MinLength != 0 && prop.MaxLength != 0 {
				if prop.MinLength > prop.MaxLength {
					return errors.New("minLength > maxLength in prop: '" + pName + "'")
				}
			}
			props[pName] = prop
		case PropDate:
			prop.clearStringParams()
			prop.clearRefParams()
			prop.clearObjectParams()
			if prop.Default != nil {
				_, ok := prop.Default.(time.Time)
				if !ok {
					return errors.New("Wrong default date in prop: '" + pName + "'")
				}
			}
			props[pName] = prop
		case PropRef:
			prop.clearStringParams()
			prop.clearNumberParams()
			prop.clearObjectParams()
			if prop.Default != nil {
				if _, ok := prop.Default.(string); !ok {
					return errors.New("Wrong default value for ref type in prop: '" + pName + "'")
				}
			}
			prop.Default = nil
			if prop.Store == "" {
				return errors.New("Store not provided for ref type in prop: '" + pName + "'")
			}
			if prop.PopulateIn != "" {
				if _, ok := m.Props[prop.PopulateIn]; ok {
					return errors.New("PopulateIn points to existing prop in ref prop: '" + pName + "'")
				}
			}
			props[pName] = prop
		case PropRefList:
			prop.clearStringParams()
			prop.clearNumberParams()
			prop.clearObjectParams()
			if prop.Default != nil {
				if _, ok := prop.Default.([]interface{}); !ok {
					return errors.New("Wrong default value for refList type in prop: '" + pName + "'")
				}
			}
			if prop.Store == "" {
				return errors.New("Store not provided for refList type in prop: '" + pName + "'")
			}
			if prop.PopulateIn != "" {
				if _, ok := m.Props[prop.PopulateIn]; ok {
					return errors.New("PopulateIn points to existing prop in refList prop: '" + pName + "'")
				}
			}
			props[pName] = prop
		case PropVirtual:
			prop.clearStringParams()
			prop.clearRefParams()
			prop.clearNumberParams()
			prop.clearObjectParams()
			prop.Default = nil
			if m.Type != ObjWorkspace {
				virtualPropsLoaders[pName] = `(function($item, $user){` + prop.Load + `})`
			}
			props[pName] = prop
		case PropObject:
			if !parseObjects {
				return errors.New("Recursive objects not allowed '" + pName + "'")
			}
			prop.clearStringParams()
			prop.clearRefParams()
			prop.clearNumberParams()
			//			prop.Default = nil
			err := m.validateProps(prop.Props, false)
			if err != nil {
				return err
			}
			props[pName] = prop
		case PropObjectList:
			if !parseObjects {
				return errors.New("Recursive objects not allowed '" + pName + "'")
			}
			prop.Pattern = ""
			prop.Mask = ""
			prop.clearRefParams()
			prop.clearNumberParams()
			err := m.validateProps(prop.Props, false)
			if err != nil {
				return err
			}
			props[pName] = prop
		case PropVirtualRefList:
			prop.clearStringParams()
			prop.clearNumberParams()
			prop.clearObjectParams()
			prop.Default = nil
			if prop.Store == "" {
				return errors.New("Store not provided for virtualRefList type in prop: '" + pName + "'")
			}
			if prop.ForeignKey == "" {
				return errors.New("Foregn key not provided for virtualRefList type in prop: '" + pName + "'")
			}
			props[pName] = prop
		case PropComments:
			prop.clearStringParams()
			prop.clearNumberParams()
			props[pName] = prop
		case PropVirtualClient:
		default:
			return errors.New("Unknown prop type: '" + pName + "' '" + prop.Type + "'")
		}
	}
	if len(virtualPropsLoaders) > 0 {
		script := "function($item, $user){\n"
		for k, v := range virtualPropsLoaders {
			script += `$item.` + k + ` = ` + v + "($item, $user);\n"
		}
		script += "return $item\n}\n"
		VirtualPropsLoaders[m.Store] = script
		m.HasVirtualProps = true
	}
	return nil
}

func (m *Model) checkPropsRequiredConditions() {
	for k, v := range m.Props {
		v.createRequired()
		m.Props[k] = v
	}
}

func (p *Prop) createRequired() {
	if p.Required == nil {
		return
	}

	if r, ok := p.Required.(bool); ok {
		p.requiredBool = r
		return
	}

	return

	encoded, err := json.Marshal(p.Required)
	if err != nil {
		logger.Error("Can't marshal required", p.Required, err.Error())
		return
	}
	var required []*Condition
	err = json.Unmarshal(encoded, &required)
	if err != nil {
		logger.Error("Can't unmarshal required", string(encoded), err.Error())
		return
	}
	p.requiredConditions = required
}

func (p *Prop) checkDefaultFloat() (float64, bool) {
	_def, ok := p.Default.(float64)
	if !ok {
		return 0, false
	}
	return _def, true
}

func (p *Prop) checkDefaultInt() (int, bool) {
	_def, ok := p.checkDefaultFloat()
	if !ok {
		return 0, false
	}
	def := int(_def)
	return def, true
}

func (p *Prop) checkMinMaxParams() (float64, float64, bool) {
	var min, max float64
	if p.Min != nil {
		var ok bool
		min, ok = p.Min.(float64)
		if !ok {
			return 0, 0, false
		}
	}
	if p.Max != nil {
		var ok bool
		max, ok = p.Max.(float64)
		if !ok {
			return 0, 0, false
		}
	}
	if min == 0 && max == 0 {
		return min, max, true
	}
	if min > max {
		return 0, 0, false
	}
	return min, max, true
}

func (p *Prop) clearNumberParams() {
	p.Min = nil
	p.Max = nil
}

func (p *Prop) clearObjectParams() {
	p.Props = nil
}

func (p *Prop) clearRefParams() {
	p.Store = ""
	p.PopulateIn = ""
}

func (p *Prop) clearStringParams() {
	p.MinLength = 0
	p.MaxLength = 0
	p.Pattern = ""
	p.Mask = ""
}

func (m *Model) LoadDefaultIntoProp(name string, p Prop) {
	if m.Props == nil {
		m.Props = map[string]Prop{}
	}
	if !p.Configurable {
		m.Props[name] = p
		return
	}

	prop, ok := m.Props[name]
	if !ok {
		m.Props[name] = p
		return
	}

	if prop.Type != "" {
		p.Type = prop.Type
	}
	if prop.FormGroup != "" {
		p.FormGroup = prop.FormGroup
	}
	if prop.FormTab != "" {
		p.FormTab = prop.FormTab
	}
	if prop.FormOrder != 0 {
		p.FormOrder = prop.FormOrder
	}
	if prop.Access != nil {
		p.Access = prop.Access
	}
	if prop.Display != "" {
		p.Display = prop.Display
	}
	// TODO придумать как поступать с булевыми полями. Если оно отсутствует в JSON, то всегда будет false
	p.ReadOnly = prop.ReadOnly
	p.Required = prop.Required

	if prop.Default != nil {
		p.Default = prop.Default
	}
	if prop.MinLength != 0 {
		p.MinLength = prop.MinLength
	}
	if prop.MaxLength != 0 {
		p.MaxLength = prop.MaxLength
	}
	if prop.Min != nil {
		p.Min = prop.Min
	}
	if prop.Max != nil {
		p.Max = prop.Max
	}
	if prop.Hidden != nil {
		p.Hidden = prop.Hidden
	}
	if prop.Pattern != nil {
		p.Pattern = prop.Pattern
	}
	if prop.Mask != nil {
		p.Mask = prop.Mask
	}
	if prop.Load != "" {
		p.Load = prop.Load
	}
	if prop.Store != "" {
		p.Store = prop.Store
	}
	if prop.PopulateIn != "" {
		p.PopulateIn = prop.PopulateIn
	}
	if prop.Label != "" {
		p.Label = prop.Label
	}
	if prop.Placeholder != "" {
		p.Placeholder = prop.Placeholder
	}
	if prop.Disabled != "" {
		p.Disabled = prop.Disabled
	}

	m.Props[name] = p
}

func (m *Model) mergeAccess(defaultStore *Model) {
	if m.Access == nil {
		for i := range defaultStore.Access {
			m.Access = append(m.Access, defaultStore.Access[i])
		}
	}
}

func (m *Model) mergeFilters(defaultStore *Model) {
	if len(defaultStore.Filters) == 0 {
		return
	}
	if len(m.Filters) == 0 {
		m.Filters = map[string]Filter{}
	}
	for k, v := range defaultStore.Filters {
		f, ok := m.Filters[k]
		if !ok {
			m.Filters[k] = v
			continue
		}
		if f.Label == "" {
			f.Label = v.Label
		}
		if f.Display == "" {
			f.Display = v.Display
		}
		if f.Placeholder == "" {
			f.Placeholder = v.Placeholder
		}
		if len(f.Conditions) == 0 {
			f.Conditions = v.Conditions
		}
		if len(f.SearchBy) == 0 {
			f.SearchBy = v.SearchBy
		}
		if f.Store == "" {
			f.Store = v.Store
		}
		if f.FilterBy == "" {
			f.FilterBy = v.FilterBy
		}
		if len(f.Options) == 0 {
			f.Options = v.Options
		}
		if f.Mask == "" {
			f.Mask = v.Mask
		}
		if !f.Multi {
			f.Multi = v.Multi
		}
		m.Filters[k] = f
	}
}

func mergeModels(from, to *Model) {
	if from.Filters != nil {
		to.Filters = from.Filters
	}
	if from.NavGroup != "" {
		to.NavGroup = from.NavGroup
	}
	if from.FormGroupsOrder != nil {
		to.FormGroupsOrder = from.FormGroupsOrder
	}
	if len(from.I18n) > 0 {
		mergo.MergeWithOverwrite(&to.I18n, from.I18n)
	}
	if len(from.Entries) > 0 {
		mergo.MergeWithOverwrite(&to.Entries, from.Entries)
	}
	if from.NavOrder != 0 {
		to.NavOrder = from.NavOrder
	}
	if from.Display != "" {
		to.Display = from.Display
	}
	if from.Icon != "" {
		to.Icon = from.Icon
	}
	if from.PrepareItemsScript != "" {
		to.PrepareItemsScript = from.PrepareItemsScript
	}
	if len(from.Labels) > 0 {
		to.Labels = from.Labels
	}
	if len(from.TableColumns) > 0 {
		to.TableColumns = from.TableColumns
	}
	if from.OrderBy != "" {
		to.OrderBy = from.OrderBy
	}
	if from.Html != "" {
		to.Html = from.Html
	}
	if from.Label != "" {
		to.Label = from.Label
	}
	if from.NavLabel != "" {
		to.NavLabel = from.NavLabel
	}
	if from.Template != "" {
		to.Template = from.Template
	}
	if from.TemplateFile != "" {
		to.TemplateFile = from.TemplateFile
	}
	if from.ListViewOnly != false {
		to.ListViewOnly = from.ListViewOnly
	}
	if len(from.TableColumns) > 0 {
		to.TableColumns = from.TableColumns
	}
	if len(from.Actions) > 0 {
	FromActionsLoop:
		for _, v := range from.Actions {
			for tk, tv := range to.Actions {
				if v.Id == tv.Id {
					if v.Label != "" {
						tv.Label = v.Label
					}
					to.Actions[tk] = tv
					continue FromActionsLoop
				}
			}
			to.Actions = append(to.Actions, v)
		}
	}
	for k, vFrom := range from.Props {
		if vTo, ok := to.Props[k]; ok {
			mergeProps(&vFrom, &vTo)
			to.Props[k] = vTo
			continue
		}
		if to.Store == "_profile" {
			switch vFrom.Type {
			case PropVirtual, PropVirtualClient, PropAction:
				to.Props[k] = vFrom
			}
		}
	}
}

func mergeProps(from, to *Prop) {
	if from.TableLink {
		to.TableLink = from.TableLink
	}
	if from.FormGroup != "" {
		to.FormGroup = from.FormGroup
	}
	if from.FormOrder != 0 {
		to.FormOrder = from.FormOrder
	}
	if from.Display != "" {
		to.Display = from.Display
	}
	if from.Html != "" {
		to.Html = from.Html
	}
	if from.DisplayWidth != 0 {
		to.DisplayWidth = from.DisplayWidth
	}
	if from.Style != nil {
		to.Style = from.Style
	}
	if from.Default != nil {
		to.Default = from.Default
	}
	if from.Pattern != "" { //TODO: Must be compile
		to.Pattern = from.Pattern
	}
	if from.Mask != "" {
		to.Mask = from.Mask
	}
	if from.Placeholder != "" {
		to.Placeholder = from.Placeholder
	}
	if from.MaxLength != 0 {
		to.MaxLength = from.MaxLength
	}
	if from.MinLength != 0 {
		to.MinLength = from.MinLength
	}
	if len(from.Options) > 0 {
		to.Options = from.Options
	}
	if from.Hidden != nil {
		to.Hidden = from.Hidden
	}
	if from.Disabled != nil {
		to.Disabled = from.Disabled
	}
	if from.Required != nil {
		to.Required = from.Required
	}
	if from.FormTab != "" {
		to.FormTab = from.FormTab
	}
	for k, vFrom := range from.Props {
		if vTo, ok := to.Props[k]; ok {
			mergeProps(&vFrom, &vTo)
			to.Props[k] = vTo
		}
	}
}

func (m *Model) LoadDefaultValues(data bdb.M) {
	for k, v := range m.Props {
		if v.Default == nil {
			continue
		}
		_v, ok := data[k]
		if !ok || _v == nil {
			data[k] = v.Default
		} else {
			kind := reflect.TypeOf(_v).Kind()
			switch kind {
			case reflect.Array, reflect.Slice, reflect.Map, reflect.Interface:
				if reflect.ValueOf(_v).IsNil() {
					data[k] = v.Default
				}
			}
		}
	}
}

func (m *Model) preparePartialFlags() {
	m.PartialProps = []string{}
	if len(m.TableColumns) > 0 {
		for _, v := range m.TableColumns {
			switch v.(type) {
			case string:
				prop := v.(string)
				if _, ok := m.Props[prop]; ok || strings.Contains(prop, ".") {
					m.PartialProps = append(m.PartialProps, prop)
				}
			default:
				columnDesc, ok := v.(map[string]interface{})
				if ok {
					prop, ok := columnDesc["prop"].(string)
					if ok {
						if _, ok := m.Props[prop]; ok || strings.Contains(prop, ".") {
							m.PartialProps = append(m.PartialProps, prop)
						}
					}
				}
			}
		}
	}
	if m.Type == ObjProcess {
		if array.InArrayString(m.PartialProps, "_state") == -1 {
			m.PartialProps = append(m.PartialProps, "_state")
		}
	}
	if m.OrderBy != "" {
		if array.InArrayString(m.PartialProps, m.OrderBy) == -1 {
			m.PartialProps = append(m.PartialProps, m.OrderBy)
		}
	}
	if m.Type == ObjNotification {
		for k := range m.Props {
			if array.InArrayString(m.PartialProps, k) == -1 {
				m.PartialProps = append(m.PartialProps, k)
			}
		}
	}
	if match := mustacheRgx.FindAllStringSubmatch(m.Html, -1); len(match) > 0 {
		for _, str := range match {
			if subMatch := handleBarseRgx.FindAllStringSubmatch(str[1], -1); len(subMatch) > 0 && len(subMatch[0]) > 2 {
				var prop string
				if subMatch[0][2] == "" {
					prop = subMatch[0][1]
				} else {
					prop = subMatch[0][2]
				}
				if array.InArrayString(m.PartialProps, prop) == -1 {
					if _, ok := m.Props[prop]; ok || strings.Contains(prop, ".") {
						m.PartialProps = append(m.PartialProps, prop)
					}
				}
			}
		}
	}
	headerTemplateProps := itemPropsRgx.FindAllStringSubmatch(m.HeaderTemplate, -1)
	for _, prop := range headerTemplateProps {
		if array.InArrayString(m.PartialProps, prop[1]) == -1 {
			m.PartialProps = append(m.PartialProps, prop[1])
		}
	}
	if array.InArrayString(m.PartialProps, m.HeaderProperty) == -1 {
		m.PartialProps = append(m.PartialProps, m.HeaderProperty)
	}
	for _, v := range m.Labels {
		if v.ShowInList > 0 {
			labelProps := itemPropsRgx.FindAllStringSubmatch(v.Text, -1)
			for _, prop := range labelProps {
				if array.InArrayString(m.PartialProps, prop[1]) == -1 {
					m.PartialProps = append(m.PartialProps, prop[1])
				}
			}
			labelProps = itemPropsRgx.FindAllStringSubmatch(v.Icon, -1)
			for _, prop := range labelProps {
				if array.InArrayString(m.PartialProps, prop[1]) == -1 {
					m.PartialProps = append(m.PartialProps, prop[1])
				}
			}
			labelProps = itemPropsRgx.FindAllStringSubmatch(v.Color, -1)
			for _, prop := range labelProps {
				if array.InArrayString(m.PartialProps, prop[1]) == -1 {
					m.PartialProps = append(m.PartialProps, prop[1])
				}
			}
			labelProps = itemPropsRgx.FindAllStringSubmatch(v.Hidden, -1)
			for _, prop := range labelProps {
				if array.InArrayString(m.PartialProps, prop[1]) == -1 {
					m.PartialProps = append(m.PartialProps, prop[1])
				}
			}
		}
	}
	for _, v := range m.PartialProps {
		if p, ok := m.Props[v]; ok {
			if p.Type == PropVirtual {
				m.PartialVirtual = true
				if !m.PartialPopulate {
					virtualScriptProps := itemPropsRgx.FindAllStringSubmatch(p.load, -1)
					for _, prop := range virtualScriptProps {
						if _, ok := m.Props[prop[1]]; !ok {
							m.PartialPopulate = true
							break
						}
					}
				}
			}
		} else {
			m.PartialPopulate = true
		}
	}
}

func (m *Model) prepareI18nForUser(u User) {
	if u.GetLanguage() != "" {
		if _locale, ok := m.I18n[u.GetLanguage()]; ok {
			if locale, ok := _locale.(map[string]interface{}); ok {
				m.I18n = locale
				return
			}
		}
	}
	if _locale, ok := m.I18n[CommonSettings.DefaultLocale]; ok {
		if locale, ok := _locale.(map[string]interface{}); ok {
			m.I18n = locale
			return
		}
	}
	if _locale, ok := m.I18n["en"]; ok {
		if locale, ok := _locale.(map[string]interface{}); ok {
			m.I18n = locale
			return
		}
	}
}
