#!/usr/bin/perl

use strict;
use DBI;
use Data::Dumper;

use lib qw(..);
use Genelet::DBI;
use Goto::Config;

my $dbh = DBI->connect(@{$Goto::Config::controller{db}});
my $form = Genelet::DBI->new(dbh=>$dbh);
$form->do_sql("SET NAMES 'utf8'");

my $lists = [];
my $ref;
my @items;
my $err;

$err = $form->select_sql($lists,
	"SELECT continent_id,continent_code FROM def_continent") and die $err;
for (@$lists) {
	$ref->{$_->{continent_code}} = $_->{continent_id};
}

my $country;
open(C, "newiso31661") || die $!;
while(my $line = <C>) {
	chomp $line;
	my @a = split /\s*\t\s*/, $line, -1;
	pop @a;
	die $line unless (@a==4);
	$country->{$a[1]} = \@a;
}
close(C);

open(C, "continent") || die $!;
while(my $line = <C>) {
	if ($line =~ /(\S\S),(\S\S)\s*$/) {
		$country->{$1}->[4] = $ref->{$2} if $country->{$1};
	} else {
		die $line;
	}
}
close(C);

for (sort keys %$country) {
	push @items, $country->{$_};
}

$err = $form->do_sqls(
	"INSERT INTO def_country 
	(country_name, country_code, alpha3, numeric_code, continent_id)
	VALUES (?,?,?,?,?)", @items) ||
	$form->do_sql(
	"update def_country set active='Yes' 
	where country_code in ('US','CN','DE','FR','GB','AU','CA','JP','RU','KR','IN','BR','IT','NL','ES')");
die $err if $err;

$lists = [];
$ref = {};
@items = ();

$err = $form->select_sql($lists,
	"SELECT country_id, country_code FROM def_country");
die $err if $err;
for (@$lists) {
	$ref->{$_->{country_code}} = $_->{country_id};
}

open(C, "newiso31662") || die $!;
while(my $line = <C>) {
	my @a = split /\s*\t\s*/, $line, -1;
	for (@a) {
		$_ =~ s/^\s+//;
		$_ =~ s/\s+$//;
	}
	next if (@a==2);
	my ($country_code, $state_code) = split /-/, $a[1];
	next unless $ref->{$country_code};
	push @items, [$ref->{$country_code}, $192950state_code, $a[2]];
}
close(C);

$err = $form->do_sqls(
	"INSERT INTO def_state (country_id, state_code, state_name) 
	VALUES (?,?,?)", @items) and die $err;

$lists = [];
$ref = {};
@items = ();

$err = $form->select_sql($lists,
	"SELECT s.state_id, s.state_code, c.country_code
	FROM def_state s 
	INNER JOIN def_country c USING (country_id)");
for (@$lists) {
	$ref->{$_->{country_code}}->{$_->{state_code}} = $_->{state_id};
}

open(C, "GeoIPCity-534-Location.csv") || die $!;
while(my $line = <C>) {
	chomp $line;
    my @b = split(',', $line, -1);
    next if ($b[2] eq '""' and $b[3] eq '""');
	for my $i (1,2,3,4) {
      $b[$i] =~ s/"//g;
    }
    next unless $ref->{$b[1]}->{$b[2]};
    push @items, [$ref->{$b[1]}->{$b[2]},$b[3],$b[4],$b[5],$b[6]];
}
close(C);

$err = $form->do_sqls(
	"INSERT INTO def_city (state_id, city_name, postal, latitude, longitude)
	VALUES (?,?,?,?,?)", @items) and return $err;

$lists = [];
$ref = {};
@items = ();
$err = $form->select_sql($lists,
	"SELECT s.state_id, s.state_name
	FROM def_state s
	INNER JOIN def_country c USING (country_id)
	WHERE c.country_code='US'");
for (@$lists) {
	$ref->{$_->{state_name}} = $_->{state_id};
}

open(C, "metrocodes.csv") || die $!;
while(my $line = <C>) {
	chomp $line;
	my ($name, $description, $metrocode) = split /"\,"/, $line;
	$name    =~ s/^"//;
	$metrocode =~ s/"$//;
	next unless $ref->{$name};
	push @items, [$ref->{$name}, $metrocode, $description];
}
close(C);

$err = $form->do_sqls(
	"INSERT INTO def_dma (state_id, metro_code, description)
	VALUES (?,?,?)", @items) and return $err;


$dbh->disconnect;
exit;
