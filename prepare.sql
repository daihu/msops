CREATE USER 'dba'@'%' IDENTIFIED BY 'dba';
GRANT RELOAD, PROCESS, SUPER, REPLICATION CLIENT, REPLICATION SLAVE ON *.* TO 'dba'@'%';
CREATE USER 'repl'@'%' IDENTIFIED BY 'repl';
GRANT PROCESS, REPLICATION SLAVE ON *.* TO 'dba'@'%';
CREATE DATABASE data_test;
GRANT ALL ON data_test.* TO 'dba'@'%';
USE data_test;
CREATE TABLE tbl_test (
    id int primary key,
    name varchar(20) NOT NULL
);
INSERT INTO TABLE `tbl_test` VALUES(1, "hello");
INSERT INTO TABLE `tbl_test` VALUES(2, "world");
FLUSH PRIVILEGES;