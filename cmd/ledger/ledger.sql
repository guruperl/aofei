drop table if exists daily_pub_adv;
drop table if exists daily_adv;
drop table if exists daily_pub;
drop table if exists daily_log;
drop table if exists ledger_pub_adv;
drop table if exists ledger_adv;
drop table if exists ledger_pub;
drop table if exists ledger_log;

create table ledger_log (
	log_id int unsigned auto_increment not null,
	timely datetime not null,
	spend float default null,
	imps int unsigned default null,
	clis int unsigned default null,
	created datetime not null,
	primary key (log_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

create table ledger_pub (
	lp_id int unsigned auto_increment not null,
	log_id int unsigned,
	slot_id int unsigned not null,
	site_id int unsigned not null,
	pub_id int unsigned not null,
	spend float default null,
	imps int unsigned default null,
	clis int unsigned default null,
	primary key(lp_id),
	foreign key (log_id) references ledger_log (log_id) on update cascade 
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

create table ledger_adv (
	la_id int unsigned auto_increment not null,
	log_id int unsigned not null,
	item_id int unsigned not null,
	campaign_id int unsigned not null,
	adv_id int unsigned not null,
	spend float default null,
	imps int unsigned default null,
	clis int unsigned default null,
	primary key(la_id),
	foreign key (log_id) references ledger_log (log_id) on update cascade 
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

create table ledger_pub_adv (
	lpa_id int unsigned auto_increment not null,
	lp_id int unsigned not null,
	la_id int unsigned not null,
	spend float default null,
	imps int unsigned default null,
	clis int unsigned default null,
	primary key (lpa_id),
	foreign key (lp_id) references ledger_pub (lp_id) on update cascade,
	foreign key (la_id) references ledger_adv (la_id) on update cascade
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

create table daily_log (
	log_id int unsigned auto_increment not null,
	daily date not null,
	spend float default null,
	imps int unsigned default null,
	clis int unsigned default null,
	created datetime not null,
	primary key (log_id),
	index (daily)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

create table daily_pub (
	lp_id int unsigned auto_increment not null,
	log_id int unsigned,
	slot_id int unsigned not null,
	site_id int unsigned not null,
	pub_id int unsigned not null,
	spend float default null,
	imps int unsigned default null,
	clis int unsigned default null,
	primary key(lp_id),
	foreign key (log_id) references daily_log (log_id) on update cascade 
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

create table daily_adv (
	la_id int unsigned auto_increment not null,
	log_id int unsigned not null,
	item_id int unsigned not null,
	campaign_id int unsigned not null,
	adv_id int unsigned not null,
	spend float default null,
	imps int unsigned default null,
	clis int unsigned default null,
	primary key(la_id),
	foreign key (log_id) references daily_log (log_id) on update cascade 
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

create table daily_pub_adv (
	lpa_id int unsigned auto_increment not null,
	lp_id int unsigned not null,
	la_id int unsigned not null,
	spend float default null,
	imps int unsigned default null,
	clis int unsigned default null,
	primary key (lpa_id),
	foreign key (lp_id) references daily_pub (lp_id) on update cascade,
	foreign key (la_id) references daily_adv (la_id) on update cascade
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
