insert into adv_attrname (attrname_id, attrname) values (1010, "language");
insert into adv_attrname (attrname_id, attrname) values (1109, "lat");
insert into adv_attrname (attrname_id, attrname) values (1110, "lon");
insert into adv_attrname (attrname_id, attrname) values (1112, "utcoffset");
delete from adv_attrname where attrname_id between 1 and 99;
delete from adv_attrname where attrname_id between 1004 and 1006;
delete from adv_attrname where attrname_id between 1401 and 1416;
insert into adv_attrname (attrname_id, attrname) values (1400, 'uploads');
