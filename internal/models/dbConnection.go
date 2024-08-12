package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"go_final_project/internal/utils"
)

// Структура DBConnection отвечает за взаимодействие с базой данных для выполнения операций CRUD
// с задачами. Она предоставляет методы для инициализации базы данных, проверки существования задачи по ID,
// удаления задачи, вставки новой задачи, обновления существующей задачи, обновления даты существующей задачи,
// получения всех задач, получения определенной задачи по ID, получения задач, содержащих определенное ключевое слово
// в заголовке или комментарии, и получения задач, запланированных на определенную дату.
type DBConnection struct {
	db     *sql.DB
	logger *zap.SugaredLogger
}

// NewConnection создает новый экземпляр DBConnection с указанным подключением к базе данных и логгером.
//
// Параметры:
// db - Указатель на объект sql.DB, представляющий подключение к базе данных.
// logger - Указатель на объект zap.SugaredLogger для ведения журнала.
//
// Возвращает:
// Указатель на объект DBConnection, который инкапсулирует подключение к базе данных и логгер.
func NewConnection(db *sql.DB, logger *zap.SugaredLogger) *DBConnection {
	return &DBConnection{
		db:     db,
		logger: logger,
	}
}

// InitDB инициализирует базу данных путем создания таблицы "scheduler" и индекса на столбце "date".
// Таблица "scheduler" имеет следующие столбцы:
// - id: первичный ключ целого числа с автоинкрементом
// - date: поле символьного типа длиной 8 символов, не может быть NULL, имеет значение по умолчанию пустая строка
// - title: поле переменной длины строки с максимальной длиной 128 символов, не может быть NULL, имеет значение по умолчанию пустая строка
// - comment: текстовое поле для хранения комментариев
// - repeat: поле переменной длины строки с максимальной длиной 128 символов, не может быть NULL, имеет значение по умолчанию пустая строка
//
// Если во время выполнения SQL-запросов возникает ошибка, она будет выведена в консоль.
func (c *DBConnection) InitDB() error {
	const (
		CreateTableQuery = `CREATE TABLE scheduler (
		id      INTEGER PRIMARY KEY AUTOINCREMENT,
		date    CHAR(8) NOT NULL DEFAULT "",
		title   VARCHAR(128) NOT NULL DEFAULT "",
		comment TEXT,
		repeat VARCHAR(128) NOT NULL DEFAULT "" 
		);`
	)
	if _, err := c.db.Exec(CreateTableQuery); err != nil {
		return err
	}

	if _, err := c.db.Exec(`CREATE INDEX taks_date ON scheduler (date);`); err != nil {
		return err
	}
	return nil
}

// CheckID проверяет, существует ли указанный идентификатор в базе данных.
// Он извлекает максимальный идентификатор из таблицы "scheduler" и сравнивает его с указанным идентификатором.
// Если указанный идентификатор больше максимального идентификатора, возвращается ошибка, указывающая, что идентификатор больше числа строк в базе данных.
//
// Параметры:
// - id: Проверяемый идентификатор. Это целое число.
//
// Возвращает:
// - Ошибку, если указанный идентификатор больше максимального идентификатора в базе данных.
// - nil, если указанный идентификатор существует в базе данных.
func (c *DBConnection) CheckID(id int) error {
	var maxID int
	row := c.db.QueryRow(`SELECT MAX(id) FROM scheduler`)
	row.Scan(&maxID)
	if err := row.Err(); err != nil {
		return err
	}
	if id > maxID {
		err := errors.New("given id is more than number of rows in DB")
		return err
	}
	return nil
}

// Delete удаляет задачу из базы данных на основе указанного идентификатора.
//
// Параметры:
// - id: Уникальный идентификатор удаляемой задачи.
//
// Возвращает:
// - Ошибку, если во время удаления произошла ошибка. Если удаление выполнено успешно, возвращается nil
func (c *DBConnection) Delete(id int) error {
	_, err := c.db.Exec(`DELETE FROM scheduler WHERE id = ?`, id)
	if err != nil {
		c.logger.Errorw("Error deleting task", "error", err)
		return err
	}
	c.logger.Infof("Task with ID: %d was deleted", id)
	return nil
}

// Insert вставляет новую задачу в базу данных.
//
// Параметры:
// - task: Структура, содержащая данные новой задачи.
//
// Возвращает:
// - Идентификатор вставленной задачи и ошибку, если во время вставки произошла ошибка.
// - Если вставка выполнена успешно, возвращается идентификатор вставленной задачи и nil.
func (c *DBConnection) Insert(task *Task) (int, error) {
	res, err := c.db.Exec(`INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)`,
		task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		c.logger.Errorw("Error inserting task", "error", err)
		return 0, nil
	}
	id, err := res.LastInsertId()
	if err != nil {
		c.logger.Errorw("Error getting last insert id", "error", err)
		return 0, err
	}
	c.logger.Infof("Task inserted with ID: %d", id)
	return int(id), nil
}

// Update обновляет данные существующей задачи в базе данных.
//
// Параметры:
// - task: Структура, содержащая обновленные данные задачи.
//
// Возвращает:
// - Ошибку, если во время обновления произошла ошибка. Если обновление выполнено успешно, возвращается nil.
func (c *DBConnection) Update(task *Task) error {
	_, err := c.db.Exec(`UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat = ? WHERE id = ?`,
		task.Date, task.Title, task.Comment, task.Repeat, task.ID)
	if err != nil {
		return err
	}
	c.logger.Infof("Task `%s` updated", task.Title)
	return nil
}

// UpdateDate обновляет дату существующей задачи в базе данных.
//
// Параметры:
// - task: Структура, содержащая обновленные данные даты задачи.
//
// Возвращает:
// - Ошибку, если во время обновления произошла ошибка. Если обновление выполнено успешно, возвращается nil.
func (c *DBConnection) UpdateDate(task *Task) error {
	_, err := c.db.Exec(`UPDATE scheduler SET date = ? WHERE id = ?`,
		task.Date, task.ID)
	if err != nil {
		return err
	}
	c.logger.Infof("date of task `%s` updated", task.Title)
	return nil
}

// GetAll извлекает все задачи из базы данных с ограничением на количество возвращаемых записей.
//
// Параметры:
// - limit: Максимальное количество возвращаемых записей.
//
// Возвращает:
// - Словарь, содержащий массив задач и ошибку, если во время извлечения произошла ошибка.
// - Если извлечение выполнено успешно, возвращается словарь с массивом задач и nil.
func (c *DBConnection) GetAll(limit int) (map[string][]Task, error) {
	tasks := make(map[string][]Task)
	rows, err := c.db.Query(`SELECT id, date, title, comment, repeat FROM scheduler
	ORDER BY date LIMIT :limit`,
		sql.Named("limit", limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		task := Task{}
		err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			return nil, err
		}
		tasks["tasks"] = append(tasks["tasks"], task)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if tasks["tasks"] == nil {
		tasks["tasks"] = []Task{}
	}
	return tasks, nil
}

// GetTask извлекает конкретную задачу из базы данных на основе указанного идентификатора.
//
// Параметры:
// - id: Уникальный идентификатор извлекаемой задачи.
//
// Возвращает:
// - Указатель на структуру, содержащую данные извлеченной задачи и ошибку, если во время извлечения произошла ошибка.
// - Если извлечение выполнено успешно, возвращается указатель на структуру с данными задачи и nil.
func (c *DBConnection) GetTask(id int) (*Task, error) {
	// Получаем задачу по идентификатору
	task := Task{}
	rows, err := c.db.Query(`SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?`, id)
	if err != nil {
		return &task, err
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			return nil, err
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if task.Date == "" && task.Title == "" && task.Repeat == "" && task.Comment == "" {
		return nil, fmt.Errorf("no rows for id %d", id)
	}
	return &task, nil
}

// GetByWord извлекает все задачи, содержащие указанное ключевое слово в заголовке или комментарии,
// с ограничением на количество возвращаемых записей.
//
// Параметры:
// - key: Ключевое слово для поиска.
// - limit: Максимальное количество возвращаемых записей.
//
// Возвращает:
// - Словарь, содержащий массив задач и ошибку, если во время извлечения произошла ошибка.
// - Если извлечение выполнено успешно, возвращается словарь с массивом задач и nil.
func (c *DBConnection) GetByWord(key string, limit int) (map[string][]Task, error) {
	tasks := make(map[string][]Task)
	rows, err := c.db.Query(`SELECT id, date, title, comment, repeat FROM scheduler
	WHERE title LIKE :search OR comment LIKE :search ORDER BY date LIMIT :limit`,
		sql.Named("search", "%"+key+"%"),
		sql.Named("limit", limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		task := Task{}
		err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			return nil, err
		}
		tasks["tasks"] = append(tasks["tasks"], task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if tasks["tasks"] == nil {
		tasks["tasks"] = []Task{}
	}
	return tasks, nil
}

// GetByDate извлекает все задачи, запланированные на указанную дату, с ограничением на количество возвращаемых записей.
//
// Параметры:
// - date: Дата для поиска.
// - limit: Максимальное количество возвращаемых записей.
//
// Возвращает:
// - Словарь, содержащий массив задач и ошибку, если во время извлечения произошла ошибка.
// - Если извлечение выполнено успешно, возвращается словарь с массивом задач и nil.
func (c *DBConnection) GetByDate(date string, limit int) (map[string][]Task, error) {
	tasks := make(map[string][]Task)
	dateTime, err := time.Parse("02.01.2006", date)
	if err != nil {
		return nil, err
	}
	dateFormat := dateTime.Format("20060102")
	rows, err := c.db.Query(`SELECT id, date, title, comment, repeat FROM scheduler
		WHERE date = :date LIMIT :limit`,
		sql.Named("date", dateFormat),
		sql.Named("limit", limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		task := Task{}
		err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			return nil, err
		}
		tasks["tasks"] = append(tasks["tasks"], task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if tasks["tasks"] == nil {
		tasks["tasks"] = []Task{}
	}
	return tasks, nil
}

// Search ищет задачи по указанному ключу (слово или дата).
// Сначала он пытается разобрать ключ как дату с использованием формата "02.01.2006".
// Если это успешно, он вызывает метод GetByDate хранилища Task для получения задач по дате.
// Если ключ не может быть разобран как дата, он вызывает метод GetByWord хранилища Task для получения задач по слову.
// Если какой-либо из методов возвращает ошибку, он регистрирует ошибку с использованием предоставленного журнала и возвращает nil, error.
// Если ключ "tasks" в возвращенном словаре равен nil, он инициализирует его пустым массивом Task.
// Наконец, он возвращает словарь задач и nil.
func (c *DBConnection) Search(key string, limit int) (map[string][]Task, error) {
	const srchFormat = "02.01.2006"
	_, err := time.Parse(srchFormat, key)
	var tasks map[string][]Task
	if err != nil {
		tasks, err = c.GetByWord(key, limit)
		if err != nil {
			c.logger.Error(err)
			return nil, err
		}
	} else {
		tasks, err = c.GetByDate(key, limit)
		if err != nil {
			c.logger.Error(err)
			return nil, err
		}
	}
	if tasks["tasks"] == nil {
		tasks["tasks"] = []Task{}
	}
	return tasks, nil
}

// Done помечает задачу как выполненную и выполняет дополнительные действия.
// Если задача повторяется, она вычисляет дату следующего повторения и обновляет ее в хранилище.
// Если дата следующего повторения совпадает с текущей датой, она вычисляет новую дату повторения
// исходя из указанного интервала повторения и обновляет ее в хранилище.
// Если задача не повторяется, она удаляется из хранилища.
//
// Параметры:
// id - идентификатор задачи, которую необходимо пометить как выполненную.
//
// Возвращает:
// error - возвращает ошибку, если она возникла во время выполнения операции, или nil, если операция выполнена успешно.
func (c *DBConnection) Done(id int) error {
	const dateFormat = "20060102"
	task, err := c.GetTask(id)
	if err != nil {
		c.logger.Error(err)
		return err
	}
	if task.Repeat != "" {
		task.Date, err = utils.NextDate(time.Now(), task.Date, task.Repeat)

		if err != nil {
			c.logger.Error(err)
			return err
		}
		if task.Date == time.Now().Format(dateFormat) {
			date, err := time.Parse(dateFormat, task.Date)
			if err != nil {
				c.logger.Error(err)
				return err
			}
			rptSlc := strings.Split(task.Repeat, " ")
			subDays, err := strconv.Atoi(rptSlc[1])
			task.Date = date.AddDate(0, 0, subDays).Format(dateFormat)
			if err != nil {
				c.logger.Error(err)
				return err
			}
		}
		c.UpdateDate(task)
		c.logger.Infof("Task `%s` done", task.Title)
		return nil
	} else {
		id, err := strconv.Atoi(task.ID)
		if err != nil {
			c.logger.Error(err)
			return err
		}
		c.Delete(id)
		c.logger.Infof("Task `%s` done and deleted", task.Title)
	}
	return nil
}

func (s *DBConnection) DateToAdd(task *Task) error {
	nextDate, err := task.CompleteRequest()
	if err != nil {
		return err
	}
	task.Date = nextDate
	if task.Date != "" {
		nextDate, err = task.CheckDate()
		if err != nil {
			return err
		}
		task.Date = nextDate
	}
	return nil
}
