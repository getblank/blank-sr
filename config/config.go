// Структура любого объекта в базе данных

package config

import (
	"sync"

	"github.com/getblank/blank-sr/bdb"
)

const (
	ObjDirectory        = "directory"
	ObjProcess          = "process"
	ObjMap              = "map"
	ObjWorkspace        = "workspace"
	ObjCampaign         = "campaign"
	ObjNotification     = "notification"
	ObjSingle           = "single"
	ObjFile             = "file"
	ObjProxy            = "proxy"
	SinglesBucket       = "_singles"
	DeletedBucket       = "_deleted"
	TempFileStoreBucket = "_tmpFiles"
	ApiKeysBucket       = "apikeys"
	Deleted             = "_deleted"
	DefaultDirectory    = "_default"
	DefaultProcess      = "_defaultProcess"
	DefaultCampaign     = "_defaultCampaign"
	DefaultNotification = "_defaultNotifications"
	DefaultSingle       = "_defaultSingle"
	ObjClientSettings   = "_clientSettings"
	ObjServerSettings   = "_serverSettings"
	ObjCommonSettings   = "_commonSettings"
	ObjSysLog           = "syslog"
	ObjSystemHealth     = "systemHealth"

	PropInt            = "int"
	PropFloat          = "float"
	PropBool           = "bool"
	PropString         = "string"
	PropDate           = "date"
	PropRef            = "ref"
	PropRefList        = "refList"
	PropVirtual        = "virtual"
	PropVirtualClient  = "virtual/client"
	PropPassword       = "password"
	PropObject         = "object"
	PropObjectList     = "objectList"
	PropVirtualRefList = "virtualRefList"
	PropComments       = "comments"
	PropAny            = "any"
	PropAction         = "action"
	PropWidget         = "widget"
	PropFile           = "file"
	PropFileList       = "fileList"

	TemplatePdf  = "pdf"
	TemplateRtf  = "rtf"
	TemplateHtml = "html"
	TemplateTxt  = "txt"

	CreateAccess = "c"
	ReadAccess   = "r"
	UpdateAccess = "u"
	DeleteAccess = "d"

	bucket       = "__stores"
	AllUsersGuid = "11111111-1111-1111-1111-111111111111"
	RootGuid     = "00000000-0000-0000-0000-000000000000"
	UsersBucket  = "users"
	ArchiveState = "_archive"
)

var (
	PropDisplay = []string{
		"autocomplete",
		"textInput",
		"numberInput",
		"floatInput",
		"textArea",
		"searchBox",
		"select",
		"datePicker",
		"timePicker",
		"dateTimePicker",
		"colorPicker",
		"filePicker",
		"commentsEditor",
		"masked",
		"checkbox",
		"checkList",
		"password",
		"headerInput",
		"text",
		"code", // PRE
		"codeEditor",
		"link",
		"dataTable",
		"html",
		"react",
		"none",
		"",
	}

	ObjectDisplay = []string{
		"none",
		"table",
		"grid",
		"list",
		"html",
		"single",
		"",
	}

	ActionType = []string{
		"script",
		"form",
		"",
	}

	concurrentChannels = map[string]chan struct{}{}

	confLocker = &sync.RWMutex{}
	config     = map[string]Store{}

	HTTPApiEnabledStores = []Store{}
	DB                   = bdb.DB{}
)

// Store definition
type Store struct {
	Type               string                 `json:"type"`                                  // Допустимые пока варианты: directory (простые справочники), process (справочники с состояниями и действиями над объектами), workspace (воркспейсы), inConfigSet (набор каких-то значений)
	BaseStore          string                 `json:"baseStore,omitempty"`                   // Только для Type == 'proxy'. Содержит стору на которую производится проксирование.
	Access             []Access               `json:"access,omitempty"`                      // Разрешения для работы с объектом. Если не заполнено, то доступ разрешён всем
	GroupAccess        string                 `json:"groupAccess"`                           // Разрешения для работы с объектом в виде вычисленной для конкретного юзера строки (crud)
	OwnerAccess        string                 `json:"ownerAccess"`                           // Разрешения для работы с объектом в виде вычисленной для конкретного владельца строки (crud)
	NavGroup           string                 `json:"navGroup,omitempty" ws:"yes"`           // Определяет расположения в меню. Если группа не задана, ссылка на эти объекты будет выведена в первом уровне навигации
	I18n               map[string]interface{} `json:"i18n" ws:"yes"`                         // Вариации названий в браузере
	HeaderTemplate     string                 `json:"headerTemplate,omitempty"`              // Шаблон {{}} для отображения заголовка выбранного элемента
	HeaderProperty     string                 `json:"headerProperty,omitempty"`              // property для отображения заголовка выбранного элемента
	Icon               string                 `json:"icon,omitempty"`                        // Иконка для отображения в меню и в центре уведомлений
	Props              map[string]Prop        `json:"props" ws:"yes"`                        // Перечень свойств объекта
	Filters            map[string]Filter      `json:"filters" ws:"yes"`                      // Перечень фильтров сторы
	ObjectLifeCycle    Hooks                  `json:"objectLifeCycle,omitempty"`             // Хуки на события жизненного цикла объекта
	StoreLifeCycle     Hooks                  `json:"storeLifeCycle,omitempty"`              // Хуки на события жизненного цикла сторы
	FormGroups         []string               `json:"formGroups,omitempty" ws:"yes"`         // Порядок групп свойств на форме по-умолчанию
	FormTabs           []interface{}          `json:"formTabs" ws:"yes"`                     // Описание и порядок страниц на форме
	States             map[string]State       `json:"states,omitempty"`                      // Только для типа "process". Список возможных состояний
	Actions            []Action               `json:"actions,omitempty"`                     // Перечень действий над объектом
	StoreActions       []Action               `json:"storeActions,omitempty"`                // Перечень действий при поступлении внешних событий
	Widgets            []Widget               `json:"widgets,omitempty"`                     // Виджеты, используются для display:dashboard и property.type:widget
	Entries            map[string]interface{} `json:"entries,omitempty"`                     // Значения для type == 'map'
	NavOrder           int                    `json:"navOrder" ws:"yes"`                     // Порядок размещения в навигации
	Display            string                 `json:"display" ws:"yes"`                      // Вид отображения
	HTML               string                 `json:"html" ws:"yes"`                         // Шаблон для display:html
	Label              string                 `json:"label,omitempty"`                       // Заголовок сторы
	NavLabel           string                 `json:"navLabel,omitempty"`                    // Название в навигации
	Labels             []Label                `json:"labels,omitempty"`                      // Лейблы
	TableColumns       []interface{}          `json:"tableColumns,omitempty" ws:"yes"`       // Только для display:dataTable
	DisableAutoSelect  bool                   `json:"disableAutoSelect,omitempty" ws:"yes"`  // Только для display:listView
	OrderBy            string                 `json:"orderBy,omitempty" ws:"yes"`            // Сортировка данных по умолчанию.
	Config             map[string]Store       `json:"config,omitempty"`                      // Конфиг, переопределяющий некоторые параметры. Только для type:workspace
	ListViewOnly       bool                   `json:"listViewOnly,omitempty" ws:"yes"`       // Параметр, определяющий отображение сторы.
	FullWidth          bool                   `json:"fullWidth,omitempty" ws:"yes"`          // Параметр отображения контента во всю ширину, когда не используется боковая панель.
	DisablePartialLoad bool                   `json:"disablePartialLoad,omitempty" ws:"yes"` // TODO: Если true, то выдавать при запросе всех объектов, все поля сразу.
	HTTPApi            bool                   `json:"httpApi,omitempty"`                     // Флаг формирования HTTP REST API для сторы
	HTTPHooks          []HTTPHook             `json:"httpHooks,omitempty"`                   // Http хуки (HTTP API).
	NavLinkStyle       bdb.M                  `json:"navLinkStyle,omitempty" ws:"yes"`       // Стили кнопки в навигации
	NavLinkActiveStyle bdb.M                  `json:"navLinkActiveStyle,omitempty" ws:"yes"` // Стили кнопки в навигации
	NavLinkHoverStyle  bdb.M                  `json:"navLinkHoverStyle,omitempty" ws:"yes"`  // Стили кнопки в навигации
	Tasks              []*Task                `json:"tasks,omitempty"`                       // Периодические задачи для сторы
	PrepareItemsScript string                 `json:"prepareItemsScript,omitempty"`          // Скрипт для подготовки данных до передачи в рендер свойства html
	Template           string                 `json:"template,omitempty"`                    // Шаблон для передачи в свойство html
	TemplateFile       string                 `json:"templateFile,omitempty"`                // Фвйл с шаблоном для передачи в свойство html
	Indexes            []interface{}          `json:"indexes,omitempty"`                     // MongoDB indexes
	Store              string                 `json:"store"`                                 // Имя сторы для хранения. Берётся из ключа мапы
	Logging            bool                   `json:"logging,omitempty"`                     // Флаг указывающий на необходимость ведения журнала действий
	HasVirtualProps    bool                   `json:"hasVirtualProps,omitempty"`             // Флаг указывающий наличие виртуальных полей в сторе
	LoadComponent      string                 `json:"loadComponent,omitempty"`               // Код компонента React для загрузки. Только для display:react.
	HideHeader         bool                   `json:"hideHeader,omitempty"`                  // Флаг указывающий на запрет отображения заголовка сторы. Только для display:react и display:html
}

// Prop definition
type Prop struct {
	Name               string          `json:"name"`                                  // Название проперти
	Label              string          `json:"label,omitempty" ws:"yes"`              // Вариации названий в браузере
	Type               string          `json:"type"`                                  // Допустимые варианты: int, float, bool, string, date, ref, virtual
	OppositeProp       string          `json:"oppositeProp,omitempty"`                // Для type = ref, соответствующая поле в противоположной сторе
	FormTab            string          `json:"formTab,omitempty" ws:"yes"`            // Определяет страницу на форме, в которой будет отрисовано поле
	FormGroup          string          `json:"formGroup,omitempty" ws:"yes"`          // Определяет группу на форме, в которой будет отрисовано поле
	FormOrder          int             `json:"formOrder" ws:"yes"`                    // Определяет порядок отрисовки на форме. Если поле в группе, определяет порядок отрисовки именно в этой группе.
	Access             []Access        `json:"access,omitempty"`                      // Разрешения для работы с полем
	GroupAccess        string          `json:"groupAccess"`                           // Разрешения для работы с полем в виде вычисленной для конкретного юзера строки (crud)
	OwnerAccess        string          `json:"ownerAccess"`                           // Разрешения для работы с полем в виде вычисленной для конкретного владельца строки (crud)
	Display            string          `json:"display" ws:"yes"`                      // text, textArea, datePicker, timePicker, dateTimePicker, masked, none
	DisplayWidth       int             `json:"displayWidth,omitempty" ws:"yes"`       // Ширина инпута в процентах для вложенных объектов с шагом в 5
	Style              bdb.M           `json:"style,omitempty" ws:"yes"`              // Внезапно: пока не определим нужный набор свойств отображения, прокину как я CSS
	ClassName          string          `json:"сlassName,omitempty"`                   // CSS класс для контейнера на форме
	LabelClassName     string          `json:"labelClassName,omitempty"`              // CSS класс для лейбла
	HTML               string          `json:"html,omitempty" ws:"yes"`               // Html for display=html
	HTMLFile           string          `json:"htmlFile,omitempty" ws:"yes"`           // Файл с шаблоном Html for display=html
	SearchBy           []string        `json:"searchBy,omitempty"`                    // Поля для поиска для элемента searchBox // TODO: сделать валидацию
	SelectedTemplate   string          `json:"selectedTemplate,omitempty"`            // Шаблон выбранного элемента для searchBox
	SortBy             string          `json:"sortBy,omitempty"`                      // Поля для сортировки, если пропа virtual // TODO: сделать валидацию
	ReadOnly           bool            `json:"readOnly,omitempty"`                    // Поле только для чтения
	Required           interface{}     `json:"required,omitempty" ws:"yes"`           // Поле является обязательным
	Default            interface{}     `json:"default,omitempty" ws:"yes"`            // Значение по умолчанию
	MinLength          int             `json:"minLength,omitempty" ws:"yes"`          // Применимо только для строк
	MaxLength          int             `json:"maxLength,omitempty" ws:"yes"`          // Применимо только для строк
	Min                interface{}     `json:"min,omitempty"`                         // Применимо только для числовых типов
	Max                interface{}     `json:"max,omitempty"`                         // Применимо только для числовых типов
	Pattern            interface{}     `json:"pattern,omitempty" ws:"yes"`            // Применимо только для строк
	PatternError       string          `json:"patternError,omitempty" ws:"yes"`       // Применимо только для строк. Ошибка которая отобразится, если введенное значение не соответствует паттерну
	Mask               interface{}     `json:"mask,omitempty" ws:"yes"`               // Применимо только для строк
	Accept             string          `json:"accept,omitempty" ws:"yes"`             // Применимо только для файлов
	AddLabel           string          `json:"addLabel,omitempty" ws:"yes"`           // Применимо только для ObjectList
	Sortable           bool            `json:"sortable,omitempty" ws:"yes"`           // Применимо только для ObjectList, перетаскивание
	TableColumns       []interface{}   `json:"tableColumns,omitempty" ws:"yes"`       // Применимо только для VirtualRefList
	Placeholder        string          `json:"placeholder,omitempty" ws:"yes"`        // Применимо для строк и числовых типов
	Load               string          `json:"load,omitempty"`                        // Функция на JS, применима к типу virtual
	LoadComponent      string          `json:"loadComponent,omitempty"`               // Функция на JS для display==='react'
	Store              string          `json:"store,omitempty"`                       // Имя сторы для поля типа ref
	ForeignKey         string          `json:"foreignKey,omitempty"`                  // Имя пропы в референсной сторе для поля типа virtualRefList
	PopulateIn         interface{}     `json:"populateIn,omitempty"`                  // Куда складывать популизованные данные. Если значения нет, то популяция не требуется
	Configurable       bool            `json:"configurable,omitempty"`                // Только для объекта _default. Если true, то админ может переопределить настроки поля
	Hidden             interface{}     `json:"hidden,omitempty" ws:"yes"`             // Hidden conditions, JavaScript expression
	Disabled           interface{}     `json:"disabled,omitempty" ws:"yes"`           // Disabled conditions, JavaScript expression
	DisableOrder       bool            `json:"disableOrder,omitempty" ws:"yes"`       // Запрет сортировки по полю
	DisableCustomInput bool            `json:"disableCustomInput,omitempty" ws:"yes"` // Запрет ручного ввода данных для некоторых типов отображения
	Tooltip            string          `json:"tooltip,omitempty" ws:"yes"`            // Тултип в формате markdown
	Props              map[string]Prop `json:"props,omitempty" ws:"yes"`              // Перечень свойств вложенного объекта
	Options            []interface{}   `json:"options,omitempty"`                     // Перечень для селектов и прочего  // TODO: валидация
	NoSanitize         bool            `json:"noSanitize,omitempty"`                  // Флаг указывающий на то, что html безопасный в пропе.
	DisableRefSync     bool            `json:"disableRefSync,omitempty"`              // Флаг, запрещающий обновление соответствующего ref поля в противоположной сторе
	TableLink          bool            `json:"tableLink,omitempty" ws:"yes"`          // Флаг обозначающий, что эта пропертя в таблице будет ссылкой
	Actions            interface{}     `json:"actions,omitempty" ws:"yes"`            // Action identifier or array of identifiers
	Widgets            interface{}     `json:"widgets,omitempty" ws:"yes"`            // Widget identifier or array of identifiers
	WidgetID           string          `json:"widgetId,omitempty" ws:"yes"`           // Widget identifier
	Utc                bool            `json:"utc,omitempty" ws:"yes"`                // Работа с датами в utc - игнорирует локальное время. Только для типа date
	Format             string          `json:"format,omitempty" ws:"yes"`             // Формат отображения. Только для типа date
	ExtraQuery         interface{}     `json:"extraQuery,omitempty"`                  // Доп Запрос для монги
	Query              interface{}     `json:"query,omitempty"`                       // Запрос для монги
}

// Filter definition
type Filter struct {
	Label       string      `json:"label,omitempty" ws:"yes"`       // Вариации названий в браузере
	Display     string      `json:"display" ws:"yes"`               // textInput, searchBox, select, masked
	Placeholder string      `json:"placeholder,omitempty" ws:"yes"` // Применимо для display:textInput
	SearchBy    []string    `json:"searchBy,omitempty"`             // display:searchBox Поля для поиска для элемента searchBox // TODO: сделать валидацию
	Store       string      `json:"store,omitempty"`                // display:searchBox Имя сторы
	FilterBy    string      `json:"filterBy,omitempty"`             // display:searchBox Имя сторы для поля типа ref
	Options     []Value     `json:"options,omitempty"`              // display:select Перечень для селектов и прочего  // TODO: валидация
	Mask        string      `json:"mask,omitempty" ws:"yes"`        // display:masked Применимо только для строк
	Multi       bool        `json:"multi" ws:"yes"`                 // display:searchBox Возможность выбора нескольких элементов
	FormOrder   int         `json:"formOrder,omitempty"`            // Порядок отображения на форме
	Style       bdb.M       `json:"style,omitempty" ws:"yes"`       // CSS
	Query       interface{} `json:"query,omitempty"`                // Запрос для монги
}

// HTTPHook definition
type HTTPHook struct {
	Uri                 string `json:"uri"`                           // URI, по которому будет доступен хук. Например, если uri=users, то хук будет http://server-address/hooks/users
	Method              string `json:"method"`                        // HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)
	Script              string `json:"script"`                        // Javascript code of hook
	ConcurentCallsLimit int    `json:"concurentCallsLimit,omitempty"` // Max concurrent calls of http hook
}

// Access definition
type Access struct {
	Role        string      `json:"role"`        // Роль, для которой определяется доступ
	Permissions string      `json:"permissions"` // Права на доступ ГРУППА|ВЛАДЕЛЕЦ (crud|rud в любом сочетании), если для группы стоит знак "-", то прав на стору нет вообще. Перед каждой буквой тоже может стоять "-".
	Condition   interface{} `json:"condition"`
}

// Action – struct for JavaScript scenarios that can be performed by users
type Action struct {
	ID                    string      `json:"_id"`
	ClassName             string      `json:"className,omitempty"`           // CSS class
	ClientPreScript       string      `json:"clientPreScript,omitempty"`     // JavaScript scenario that will run in browser before request to server
	ClientPostScript      string      `json:"clientPostScript,omitempty"`    // JavaScript scenario that will run in browser after server response
	ConcurentCallsLimit   int         `json:"concurentCallsLimit,omitempty"` // Max concurrent calls of action
	Disabled              interface{} `json:"disabled,omitempty"`            // Disabled conditions, JavaScript expression
	DisableItemReadyCheck bool        `json:"disableItemReadyCheck"`         // Enables action on modified item
	DynamicLabel          bool        `json:"dynamicLabel,omitempty"`        // If true, action lable renders on every item or user change
	Hidden                interface{} `json:"hidden,omitempty"`              // Hidden conditions, JavaScript expression
	HideInHeader          bool        `json:"hideInHeader"`                  // Hides action in item header
	ShowInList            bool        `json:"showInList"`                    // Show action in items in list view
	Icon                  string      `json:"icon" ws:"yes"`                 // Icon for label
	Label                 string      `json:"label" ws:"yes"`                // Text for label, HandleBars template
	Multi                 bool        `json:"multi"`                         // Enables action for multiple items !!!NOT IMPLEMENTED!!!
	Script                string      `json:"script,omitempty"`              // Action JavaScript scenario
	Type                  string      `json:"type,omitempty"`

	//Access
	Access      []Access `json:"access,omitempty"` // Action access rules.
	GroupAccess string   `json:"groupAccess"`      // Calculated access, not used in config
	OwnerAccess string   `json:"ownerAccess"`      // Calculated access, not used in config

	//Actions with type=form
	FormLabel   string          `json:"formLabel,omitempty" ws:"yes"` // Text form header, HandleBars template
	CancelLabel string          `json:"cancelLabel,omitempty"`        // Text in form cancel button, HandleBars template
	OkLabel     string          `json:"okLabel,omitempty"`            // Text in form submit button, HandleBars template
	Props       map[string]Prop `json:"props" ws:"yes"`               // Form properties
}

// Hooks contains JavaScript hooks
type Hooks struct {
	WillCreate string `json:"willCreate,omitempty"` // Вызывается перед сохранением нового объекта/сторы
	DidCreate  string `json:"didCreate,omitempty"`  // Вызывается после сохранения нового объекта/сторы
	WillSave   string `json:"willSave,omitempty"`   // Вызывается перед сохранением существующего объекта/сторы
	DidSave    string `json:"didSave,omitempty"`    // Вызывается после сохранения существующего объекта/сторы
	WillRemove string `json:"willRemove,omitempty"` // Вызывается перед удалением объекта/сторы
	DidRemove  string `json:"didRemove,omitempty"`  // Вызывается после удаления объекта/сторы
	DidRead    string `json:"didRead,omitempty"`    // Вызывается при запрашивании конкретного объекта из сторы. Для сторы не применимо
	DidStart   string `json:"didStart,omitempty"`   // Вызывается после старта хранилища
}

// Label definition
type Label struct {
	Text       string `json:"text,omitempty" ws:"yes"`       // Текст лейблы
	Icon       string `json:"icon,omitempty" ws:"yes"`       // Иконка слева от текста (только CSS класс)
	ShowInList int    `json:"showInList,omitempty" ws:"yes"` // Порядок отображения в списке. Если 0, не отображать
	HideInForm bool   `json:"hideInForm,omitempty" ws:"yes"` // Не показывать на основой форме
	Color      string `json:"color,omitempty" ws:"yes"`      // Цвет рамочки на форме и цвет текста в списке
	Hidden     string `json:"hidden,omitempty"`              // Hidden conditions, JavaScript expression
}

// State definition
type State struct {
	Label    string `json:"label" ws:"yes"`    // Будут ещё поля?
	NavOrder int    `json:"navOrder" ws:"yes"` // Похоже что будут
}

// Value definition
type Value struct {
	Value interface{} `json:"value"` // Значение, которое будет присвоено полю
	Label string      `json:"label"` // Лейбла, которая отображается в браузере, или будет переведена для отображения
}

// Task definition
type Task struct {
	AllowConcurrent bool   `json:"allowConcurrent,omitempty"`
	Schedule        string `json:"schedule"`
	Script          string `json:"script"`
}

// Widget definition
type Widget struct {
	ID          string        `json:"_id"`
	Type        string        `json:"type,omitempty"`
	Label       string        `json:"label" ws:"yes"`           // Лейбла в браузере
	Load        string        `json:"load" ws:"yes"`            // Серверный скрипт подготовки данных
	Render      string        `json:"render,omitempty"`         // Скрипт на JS для отрисовки
	DidLoadData string        `json:"didLoadData,omitempty"`    // Скрипт, выполняемый в браузере после загрузки данных
	Access      []Access      `json:"access,omitempty"`         // Разрешения для работы с виджетом. Если не заполнено, то доступ разрешён всем
	GroupAccess string        `json:"groupAccess"`              // Разрешения для работы с виджетом в виде вычисленной для конкретного юзера строки (crud)
	OwnerAccess string        `json:"ownerAccess"`              // Разрешения для работы с виджетом в виде вычисленной для конкретного владельца строки (crud)
	ClassName   string        `json:"className,omitempty"`      // CSS class, который требуется навесить на кнопку
	Style       bdb.M         `json:"style,omitempty" ws:"yes"` // Дополнительные CSS виджета
	Columns     []interface{} `json:"columns,omitempty"`        // Columns description for widget
}
