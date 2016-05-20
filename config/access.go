package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/getblank/blank-sr/bdb"
	"github.com/getblank/blank-sr/berror"
	"github.com/getblank/blank-sr/bjson"
	"github.com/getblank/blank-sr/utils/array"

	"github.com/ivahaev/go-logger"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

var (
	svgAttrs = []string{"accent-height", "accumulate", "additive", "alphabetic", "arabic-form", "ascent", "baseProfile", "bbox", "begin", "by", "calcMode", "cap-height", "class", "color", "color-rendering", "content", "cx", "cy", "d", "dx", "dy", "descent", "display", "dur", "end", "fill", "fill-rule", "font-family", "font-size", "font-stretch", "font-style", "font-variant", "font-weight", "from", "fx", "fy", "g1", "g2", "glyph-name", "gradientUnits", "hanging", "height", "horiz-adv-x", "horiz-origin-x", "ideographic", "k", "keyPoints", "keySplines", "keyTimes", "lang", "marker-end", "marker-mid", "marker-start", "markerHeight", "markerUnits", "markerWidth", "mathematical", "max", "min", "offset", "opacity", "orient", "origin", "overline-position", "overline-thickness", "panose-1", "path", "pathLength", "points", "preserveAspectRatio", "r", "refX", "refY", "repeatCount", "repeatDur", "requiredExtensions", "requiredFeatures", "restart", "rotate", "rx", "ry", "slope", "stemh", "stemv", "stop-color", "stop-opacity", "strikethrough-position", "strikethrough-thickness", "stroke", "stroke-dasharray", "stroke-dashoffset", "stroke-linecap", "stroke-linejoin", "stroke-miterlimit", "stroke-opacity", "stroke-width", "systemLanguage", "target", "text-anchor", "to", "transform", "type", "u1", "u2", "underline-position", "underline-thickness", "unicode", "unicode-range", "units-per-em", "values", "version", "viewBox", "visibility", "width", "widths", "x", "x-height", "x1", "x2", "xlink:actuate", "xlink:arcrole", "xlink:role", "xlink:show", "xlink:title", "xlink:type", "xml:base", "xml:lang", "xml:space", "xmlns", "xmlns:xlink", "y", "y1", "y2", "zoomAndPan"}
	svgElems = []string{"circle", "defs", "desc", "ellipse", "font-face", "font-face-name", "font-face-src", "g", "glyph", "hkern", "image", "linearGradient", "line", "marker", "metadata", "missing-glyph", "mpath", "path", "polygon", "polyline", "radialGradient", "rect", "stop", "svg", "switch", "text", "title", "tspan", "use"}
)

func (m *Store) BeforeExecAction(id string) {
	id = m.Store + "actions" + id
	ch, ok := concurrentChannels[id]
	if !ok {
		return
	}
	ch <- struct{}{}
}

func (m *Store) AfterExecAction(id string) {
	id = m.Store + "actions" + id
	ch, ok := concurrentChannels[id]
	if !ok {
		return
	}
	<-ch
}

func (m *Store) BeforeExecHttpHook(id string) {
	id = m.Store + "httpHook" + id
	ch, ok := concurrentChannels[id]
	if !ok {
		return
	}
	ch <- struct{}{}
}

func (m *Store) AfterExecHttpHook(id string) {
	id = m.Store + "httpHook" + id
	ch, ok := concurrentChannels[id]
	if !ok {
		return
	}
	<-ch
}

func (m *Store) HasReadAccessJson(data []byte, u User) bool {
	return m.hasAccessJson(data, u, ReadAccess)
}

func (m *Store) HasCreateAccess(data bdb.M, u User) bool {
	return m.hasAccess(data, u, CreateAccess)
}

func (m *Store) HasReadAccess(data bdb.M, u User) bool {
	return m.hasAccess(data, u, ReadAccess)
}

func (m *Store) HasUpdateAccess(data bdb.M, u User) bool {
	return m.hasAccess(data, u, UpdateAccess)
}

func (m *Store) HasDeleteAccess(data bdb.M, u User) bool {
	return m.hasAccess(data, u, DeleteAccess)
}

func (m *Prop) HasCreateAccess(data bdb.M, u User) bool {
	return m.hasAccess(data, u, CreateAccess)
}

func (m *Prop) HasReadAccess(data bdb.M, u User) bool {
	return m.hasAccess(data, u, ReadAccess)
}

func (m *Prop) HasUpdateAccess(data bdb.M, u User) bool {
	return m.hasAccess(data, u, UpdateAccess)
}

func (m *Prop) HasDeleteAccess(data bdb.M, u User) bool {
	return m.hasAccess(data, u, DeleteAccess)
}

func (m *Store) hasAccessJson(data []byte, u User, mode string) bool {
	if (u.GetId() == RootGuid || array.InArrayString(u.GetRoles(), RootGuid) != -1) && m.Type != ObjNotification {
		return true
	}
	if m.GroupAccess == "" && m.OwnerAccess == "" {
		return true
	}
	if m.GroupAccess == "-" {
		return false
	}
	ownerId, ok := bjson.ExtractString(data, "_ownerId")
	if ok && ownerId == u.GetId() {
		if strings.Contains(m.OwnerAccess, "-"+mode) {
			return false
		}
		return strings.Contains(m.OwnerAccess, mode)
	}
	if strings.Contains(m.GroupAccess, "-"+mode) {
		return false
	}
	return strings.Contains(m.GroupAccess, mode)
}

func (m *Store) hasAccess(data bdb.M, u User, mode string) bool {
	if (u.GetId() == RootGuid || array.InArrayString(u.GetRoles(), RootGuid) != -1) && m.Type != ObjNotification {
		return true
	}
	if m.GroupAccess == "" && m.OwnerAccess == "" {
		return true
	}
	if m.GroupAccess == "-" {
		return false
	}
	ownerId, ok := data["_ownerId"]
	if ok && ownerId.(string) == u.GetId() {
		if strings.Contains(m.OwnerAccess, "-"+mode) {
			return false
		}
		return strings.Contains(m.OwnerAccess, mode)
	}
	if strings.Contains(m.GroupAccess, "-"+mode) {
		return false
	}
	return strings.Contains(m.GroupAccess, mode)
}

func (m *Prop) hasAccess(data bdb.M, u User, mode string) bool {
	if u.GetId() == RootGuid || array.InArrayString(u.GetRoles(), RootGuid) != -1 {
		return true
	}
	if m.GroupAccess == "" && m.OwnerAccess == "" {
		return true
	}
	if m.GroupAccess == "-" {
		return false
	}
	ownerId, ok := data["_ownerId"]
	if ok && ownerId.(string) == u.GetId() {
		if strings.Contains(m.OwnerAccess, "-"+mode) {
			return false
		}
		return strings.Contains(m.OwnerAccess, mode)
	}
	if strings.Contains(m.GroupAccess, "-"+mode) {
		return false
	}
	return strings.Contains(m.GroupAccess, mode)
}

func CreateModelCreateError(store string) error {
	return createPermissionError("object", store, CreateAccess)
}

func CreateModelReadError(store string) error {
	return createPermissionError("object", store, ReadAccess)
}

func CreateModelUpdateError(store string) error {
	return createPermissionError("object", store, UpdateAccess)
}

func CreateModelDeleteError(store string) error {
	return createPermissionError("object", store, DeleteAccess)
}

func CreatePropReadError(prop string) error {
	return createPermissionError("property", prop, ReadAccess)
}

func CreatePropUpdateError(prop string) error {
	return createPermissionError("property", prop, UpdateAccess)
}

func CreatePropDeleteError(prop string) error {
	return createPermissionError("property", prop, DeleteAccess)
}

func createPermissionError(item, prop, mode string) error {
	var access string
	switch mode {
	case CreateAccess:
		access = "Create"
	case ReadAccess:
		access = "Read"
	case UpdateAccess:
		access = "Modify"
	case DeleteAccess:
		access = "Delete"
	}
	return errors.New(fmt.Sprintf("%s %s %s is not allowed", access, item, prop))
}

func createProfile(u User) Store {
	profile := Store{}
	profile.Props = map[string]Prop{}
	object, ok := GetStoreObjectFromDb("users")
	if !ok {
		logger.Warn("No users store provided!")
		return profile
	}
	if _default, ok := GetStoreObjectFromDb(DefaultDirectory); ok {
		for _pName, _prop := range _default.Props {
			object.LoadDefaultIntoProp(_pName, _prop)
		}
	}
	for k, v := range object.Props {
		if len(v.Access) == 0 {
			v.GroupAccess = "crud"
			v.OwnerAccess = "crud"
		} else {
			groupAccess, ownerAccess := profile.calcPermissions(v.Access, u)
			if groupAccess == "-" {
				continue
			}
			v.GroupAccess = groupAccess
			v.OwnerAccess = ownerAccess
		}
		v.Access = nil
		profile.Props[k] = v
	}
	profile.I18n = object.I18n
	profile.prepareI18nForUser(u)
	profile.FormGroupsOrder = object.FormGroupsOrder
	profile.Store = "_profile"
	return profile
}

func (m *Store) calcPermissions(access []Access, u User) (groupAccess, ownerAccess string) {
	if m.Type != ObjWorkspace && m.Type != ObjNotification && u.GetId() == RootGuid {
		groupAccess = "crud"
		ownerAccess = "crud"
		return
	}
	if len(access) == 0 {
		return
	}
	for _, a := range access {
		if array.InArrayString(u.GetRoles(), a.Role) != -1 || array.InArrayString(u.GetRoles(), u.GetId()) != -1 || a.Role == AllUsersGuid {
			if a.Permissions == "-" {
				groupAccess = "-"
				ownerAccess = "-"
				return
			}
			permissions := strings.SplitN(a.Permissions, "|", 2)
			groupAccess = mergePermissions(groupAccess, permissions[0])
			if len(permissions) > 1 {
				ownerAccess = mergePermissions(ownerAccess, permissions[1])
			}
		}
	}
	if groupAccess == "" && ownerAccess == "" {
		groupAccess = "-"
		ownerAccess = "-"
	}
	return
}

func (m *Store) prepareTemplate() {
	if m.Template != "" {
		m.Html = m.Template
		m.Template = ""
	}
	if m.TemplateFile != "" {
		template, err := ioutil.ReadFile("local/lib/" + m.TemplateFile)
		if err != nil {
			template, err = ioutil.ReadFile(m.TemplateFile)
		}
		if err != nil {
			logger.Error("Can't load template", m.TemplateFile, err.Error())
			m.TemplateFile = ""
			return
		}
		m.TemplateFile = ""
		m.Html = string(template)
	}
}

func (m *Store) PrepareConfigForUser(u User) {
	san := bluemonday.UGCPolicy()
	san.AllowElements(svgElems...)
	san.AllowAttrs(svgAttrs...).OnElements(svgElems...)
	if m.Html != "" {
		m.Html = san.Sanitize(m.Html)
	}
	m.prepareHooks(false)
	m.ObjectLifeCycle.WillCreate = ""
	m.ObjectLifeCycle.DidCreate = ""
	m.ObjectLifeCycle.WillSave = ""
	m.ObjectLifeCycle.DidSave = ""
	m.ObjectLifeCycle.WillRemove = ""
	m.ObjectLifeCycle.DidRemove = ""
	for i := len(m.Actions); i > 0; i-- {
		a := m.Actions[i-1]
		if len(a.Access) == 0 {
			a.GroupAccess = "crud"
			a.OwnerAccess = "crud"
		} else {
			groupAccess, ownerAccess := m.calcPermissions(a.Access, u)
			if groupAccess == "-" {
				m.Actions = append(m.Actions[:i-1], m.Actions[i:]...)
				continue
			}
			ownerAccess = mergePermissions(ownerAccess, groupAccess)
			a.GroupAccess = groupAccess
			a.OwnerAccess = ownerAccess
		}
		a.Access = nil
		if a.Script != "" {
			a.ScriptId = m.Store + "_action_" + a.Id
		}
		if a.Type != "client" {
			a.Script = ""
		}
		m.Actions[i-1] = a
	}
	for i := len(m.StoreActions); i > 0; i-- {
		a := m.StoreActions[i-1]
		if len(a.Access) == 0 {
			a.GroupAccess = "crud"
			a.OwnerAccess = "crud"
		} else {
			groupAccess, ownerAccess := m.calcPermissions(a.Access, u)
			if groupAccess == "-" {
				m.StoreActions = append(m.StoreActions[:i-1], m.StoreActions[i:]...)
				continue
			}
			ownerAccess = mergePermissions(ownerAccess, groupAccess)
			a.GroupAccess = groupAccess
			a.OwnerAccess = ownerAccess
		}
		a.Access = nil
		if a.Script != "" {
			a.ScriptId = m.Store + "_storeAction_" + a.Id
		}
		a.Script = ""
		m.StoreActions[i-1] = a
	}

	m.GroupAccess = ""
	m.OwnerAccess = ""
	if len(m.Access) == 0 {
		m.GroupAccess = "crud"
		m.OwnerAccess = "crud"
	} else {
		groupAccess, ownerAccess := m.calcPermissions(m.Access, u)
		ownerAccess = mergePermissions(ownerAccess, groupAccess)
		m.GroupAccess = groupAccess
		m.OwnerAccess = ownerAccess
		m.Access = nil
	}
	for k, v := range m.Props {
		if v.Html != "" && !v.NoSanitize {
			v.Html = san.Sanitize(v.Html)
		}
		if v.Tooltip != "" {
			v.Tooltip = san.Sanitize(string(blackfriday.MarkdownCommon([]byte(v.Tooltip))))
		}
		if v.Type != PropVirtualClient {
			v.load = v.Load
			v.Load = ""
		}
		if len(v.Access) == 0 {
			v.GroupAccess = "crud"
			v.OwnerAccess = "crud"
		} else {
			groupAccess, ownerAccess := m.calcPermissions(v.Access, u)
			if groupAccess == "-" {
				continue
			}
			ownerAccess = mergePermissions(ownerAccess, groupAccess)
			v.GroupAccess = groupAccess
			v.OwnerAccess = ownerAccess
		}
		v.Access = nil
		if v.Type == PropObject || v.Type == PropObjectList {
			for key, val := range v.Props {
				if val.Html != "" && !val.NoSanitize {
					val.Html = san.Sanitize(val.Html)
				}
				if val.Tooltip != "" {
					val.Tooltip = san.Sanitize(string(blackfriday.MarkdownCommon([]byte(val.Tooltip))))
				}
				if val.Type != PropVirtualClient {
					val.load = val.Load
					val.Load = ""
				}
				val.GroupAccess = v.GroupAccess
				val.OwnerAccess = v.OwnerAccess
				val.Access = nil
				v.Props[key] = val
			}
		}
		m.Props[k] = v
	}
}

func mergePermissions(permissions, mergedPermissions string) string {
	if strings.Contains(mergedPermissions, "-c") {
		permissions = appendOrReplaceString(permissions, "c", "-c")
		mergedPermissions = strings.Replace(mergedPermissions, "-c", "", -1)
	}
	if strings.Contains(mergedPermissions, "c") {
		permissions = appendOrReplaceString(permissions, "c", "c")
		mergedPermissions = strings.Replace(mergedPermissions, "c", "", 1)
	}
	if strings.Contains(mergedPermissions, "-r") {
		permissions = appendOrReplaceString(permissions, "r", "-r")
		mergedPermissions = strings.Replace(mergedPermissions, "-r", "", -1)
	}
	if strings.Contains(mergedPermissions, "r") {
		permissions = appendOrReplaceString(permissions, "r", "r")
		mergedPermissions = strings.Replace(mergedPermissions, "r", "", 1)
	}
	if strings.Contains(mergedPermissions, "-u") {
		permissions = appendOrReplaceString(permissions, "u", "-u")
		mergedPermissions = strings.Replace(mergedPermissions, "-u", "", -1)
	}
	if strings.Contains(mergedPermissions, "u") {
		permissions = appendOrReplaceString(permissions, "u", "u")
		mergedPermissions = strings.Replace(mergedPermissions, "u", "", 1)
	}
	if strings.Contains(mergedPermissions, "-d") {
		permissions = appendOrReplaceString(permissions, "d", "-d")
		mergedPermissions = strings.Replace(mergedPermissions, "-d", "", -1)
	}
	if strings.Contains(mergedPermissions, "d") {
		permissions = appendOrReplaceString(permissions, "d", "d")
		mergedPermissions = strings.Replace(mergedPermissions, "d", "", -1)
	}
	return permissions
}

func appendOrReplaceString(s, replace, append string) string {
	if strings.Contains(s, replace) {
		if replace != append {
			s = strings.Replace(s, replace, append, 1)
		}
	} else {
		s += append
	}
	return s
}

func getValues(data bdb.M, propString string) (val []interface{}, err error) {
	val = []interface{}{}
	if strings.ContainsRune(propString, '.') {
		props := strings.SplitN(propString, ".", 2)
		_subData, ok := data[props[0]]
		if !ok {
			return nil, berror.DbNotFound
		}
		if reflect.TypeOf(_subData).Kind() == reflect.Array {
			subData, ok := _subData.([]bdb.M)
			if !ok {
				__subData, ok := _subData.([]map[string]interface{})
				if !ok {
					return nil, berror.DbNotFound
				}
				subData = []bdb.M{}
				for _, v := range __subData {
					subData = append(subData, bdb.M(v))
				}
			}
			for _, _subData := range subData {
				_val, ok := _subData[props[1]]
				if ok {
					val = append(val, _val)
				}
			}
			return
		}
	}
	_val, ok := data[propString]
	if !ok {
		return nil, berror.DbNotFound
	}
	val = append(val, _val)
	return
}
