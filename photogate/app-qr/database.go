package appqr

import (
	"crypto/rand"
	"math"
	"math/big"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	viper.SetDefault("qr.database.driver", "sqlite")
	viper.SetDefault("qr.database.dsn", ":memory:")
}

var (
	db *gorm.DB
)

type QrRecord struct {
	ID       uint64 `gorm:"primarykey"`
	Payload  string `gorm:"size:8000"`
	Template string `gorm:"size:20"`
	Dtime    int64
	Ctime    int64
}

type QrRecordRequest struct {
	ID       string
	Payload  string
	Template string
	Dtime    int64
	Ctime    int64
	Prefix   string
}

func initDatabase() {
	if db != nil {
		return
	}

	driver := viper.GetString("qr.database.driver")
	dsn := viper.GetString("qr.database.dsn")

	var dial gorm.Dialector
	if driver == "sqlite" {
		dial = sqlite.Open(dsn)
	} else if driver == "mysql" {
		dial = mysql.Open(dsn)
	} else {
		log.Fatal().Msgf("not supported driver=%s", driver)
	}

	var err error
	db, err = gorm.Open(dial, &gorm.Config{})
	//Add config to debug ==> Logger: logger.Default.LogMode(logger.Info)
	if err != nil {
		log.Fatal().Err(err).Msg("init mysql")
	}

	db.AutoMigrate(&QrRecord{})
}

func _generateId() uint64 {
	r, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		log.Fatal().Err(err).Msg("random")
	}
	id := r.Uint64()
	if id < 1000000 {
		id += 1000000
	}
	return id
}

func addNewShortHand(payload, template string) (uint64, error) {
	sh := QrRecord{
		Payload:  payload,
		Template: template,
		Ctime:    time.Now().UnixMilli(),
		Dtime:    0,
	}

	sh.ID = _generateId()
	err := db.Create(&sh).Error
	if err != nil {
		// retry one more time
		sh.ID = _generateId()
		err = db.Create(&sh).Error
	}
	return sh.ID, err
}

func getShortHandById(id uint64) (QrRecord, error) {
	sh := QrRecord{ID: id}
	err := db.Take(&sh).Error
	return sh, err
}

func findAll(limit int, offset int, query string) ([]QrRecord, error) {
	var qrRecords []QrRecord
	err := db.Where(map[string]interface{}{"Dtime": 0}).
		Where("Payload like ? OR Template like ?", "%"+query+"%", "%"+query+"%").
		Limit(limit).
		Offset(offset).
		Order("ctime desc").
		Find(&qrRecords).
		Error
	if err != nil {
		return nil, err
	}
	return qrRecords, nil
}

func updateQrRecordById(id uint64, payload string, template string) error {
	err := db.Model(&QrRecord{}).
		Where("ID", id).
		Updates(map[string]interface{}{"Payload": payload, "Template": template}).
		Error
	return err
}

func removeQrRecordById(id uint64) error {
	err := db.Model(&QrRecord{}).
		Where(map[string]interface{}{"ID": id, "Dtime": 0}).
		Updates(map[string]interface{}{"Dtime": time.Now().UnixMilli()}).
		Error
	return err
}

func findByID(id uint64) (QrRecord, error) {
	var qrRecord QrRecord
	err := db.Where(map[string]interface{}{"ID": id, "Dtime": 0}).
		First(&qrRecord).
		Error
	return qrRecord, err
}

func parseIdToQrID(qrRecord QrRecord) QrRecordRequest {
	qrRecordRequest := QrRecordRequest{
		ID:       chunkEncode(qrRecord.ID),
		Payload:  qrRecord.Payload,
		Template: qrRecord.Template,
		Dtime:    qrRecord.Dtime,
		Ctime:    qrRecord.Ctime,
	}
	return qrRecordRequest
}
