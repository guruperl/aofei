#!/usr/bin/perl

use strict;

my $pwd = `pwd`;
chomp($pwd);
print $pwd, "\n";

local *DIR;
opendir(DIR, $pwd) || die $!;
foreach my $dir (grep {/^[a-z]/} readdir(DIR)) {
  my $this = $pwd."/$dir";
  next unless (-d $this);
  chdir $this;
  local *THIS;
  opendir(THIS, $this) || die $!;
  for my $f (grep {/^([a-z]+)\.e$/} readdir(THIS)) {
    my ($pre, $e) = split /\./, $f;
    if ($f eq 'start.e') {
      system(qw(ln -s ../start.g)) unless (-e "start.g");
    } elsif ($f eq 'end.e') {
      system(qw(ln -s ../end.g)) unless (-e "end.g");
    } elsif (-e "$pre.g") {
      next;
    } else {
      system("cp",$f,"$pre.g");
      system("perl -pi -e 's/start\.e/start\.g/' $pre.g");
      system("perl -pi -e 's/end\.e/end\.g/' $pre.g");
    }
  }
  close(THIS);
}
close(DIR);

exit;
