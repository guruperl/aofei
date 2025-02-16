# log_bin_trust_function_creators = 1 
bind-address            = 127.0.0.1,192.168.68.78

create user 'eightran'@'%'  IDENTIFIED WITH authentication_plugin BY '12pass34';
grant all privileges on gotest.* to 'eightran'@'%';
grant super on *.* to 'eightran'@'%';
flush privileges;
