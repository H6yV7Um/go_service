package models

type ToutiaoUser struct {
	Uid         	int64
	Imei        	string
	Token        	string
	Time        	int
	Updtime     	int
	Level       	int
	Type        	int
	Device_type  	int
	Location     	string
	Location_sel 	string
	Channels     	string
	Tags         	string
	Settings     	string
	Favor        	string
	Aid          	string
	Device_brand 	string
	Device_model 	string
	Device_ua    	string
	Wm           	string
	Version      	string
}

func (t *ToutiaoUser) Format() map[string]interface{} {
	return map[string]interface{} {
		"uid": t.Uid,
		"imei": t.Imei,
		"token": t.Token,
		"time": t.Time,
		"updtime": t.Updtime,
		"level": t.Level,
		"type": t.Type,
		"device_type": t.Device_type,
		"location": t.Location,
		"location_sel": t.Location_sel,
		"channels": t.Channels,
		"tags": t.Tags,
		"settings": t.Settings,
		"favor": t.Favor,
		"aid": t.Aid,
		"device_brand": t.Device_brand,
		"device_model": t.Device_model,
		"device_ua": t.Device_ua,
		"wm": t.Wm,
		"version": t.Version,
	}
}
