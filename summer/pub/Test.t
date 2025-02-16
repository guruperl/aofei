#!/usr/bin/perl

use strict;
use lib '.';
use lib '../..';
use lib '/home/perl/genelib';

use JSON;
use URI::Escape;
use HTTP::Response;
use Data::Dumper;
use Genelet::Dispatch;
use Test::More tests=>10;

use Goto::Pub::Filter;

my $project='Goto';
my $script_name ="/go.fcgi",
my $document_root="/home/www/goto";
my $role = shift;
my $tag = shift;
warn "Usage perl Test.t web e" unless $tag;
my $path_info = "/$role/$tag/pub";
my $uri = $script_name.$path_info;
my $esc = uri_escape($uri);

    local $/;
    open( my $fh, '<', "../../../conf/config.json") or die $!;
    my $json_text = <$fh>;
    close($fh);
    my $config = decode_json( $json_text );
    die "No configuration." unless $config;

my $roles = $config->{Roles};
my $chartags = $config->{Chartags};

    local $/;
    open( my $jfh, '<', "component.json") or die $!;
    my $jjson_text = <$jfh>;
    close($jfh);
    my $ref = decode_json( $jjson_text );
    die "No configuration." unless $ref;

my $f = Goto::Pub::Filter->new();
$f->actions($ref->{actions});
my $actions = $f->actions();

my %hash = (
    project=>'Goto',
    mtype=>'S',
    script_name =>$script_name,
    document_root=>$document_root);

$ENV = (
    SCRIPT_NAME  =>$script_name,
    DOCUMENT_ROOT=>$document_root,
    REMOTE_ADDR  => '127.0.0.1',
    USER_AGENT   => 'genelet_tester');

my $expects = {
	startnew => {p=>''},
	edit => {pub=>''},
	update => {pub=>''}
};

my ($output, $resp);
while (my ($action, $val) = each %$actions) {
  next unless (grep {$role eq $_} @{$val->{groups}});
  #next if ($role eq 'web' and $action ne 'startnew');
  $ENV{REQUEST_METHOD} = 'GET';
  $ENV{PATH_INFO} = $path_info;
  $ENV{QUERY_STRING} = "action=$action";

  $output = Genelet::Dispatch::run_test($config, "/home/perl/u2link/lib", [qw(Ac Pub)]);
warn $output;
  $resp = HTTP::Response->parse($output);
  is($resp->code, 200, "status code is 200");
} 

exit;

