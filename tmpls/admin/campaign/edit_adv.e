<tr><td colspan=2> <h3>Campaign Quality</h3> </td></tr>
<tr><td>Content:</td><td>
<input type=radio [% IF item.content==0 %]checked[% END %] name=content value=0><label>Bait or Deceptive</label>
<input type=radio [% IF item.content==1 %]checked[% END %] name=content value=1><label>Casino or Gambling</label>
<input type=radio [% IF item.content==2 %]checked[% END %] name=content value=2><label>Puzzle or IQ Quizzes</label>
<input type=radio [% IF item.content==3 %]checked[% END %] name=content value=3><label>Provocative or Suggestive</label>
<input type=radio [% IF item.content==4 %]checked[% END %] name=content value=4><label>Unknown Dating</label>
<input type=radio [% IF item.content==5 %]checked[% END %] name=content value=5><label>Normal</label>
<input type=radio [% IF item.content==6 %]checked[% END %] name=content value=6><label>Good Promotion</label>
<input type=radio [% IF item.content==7 %]checked[% END %] name=content value=7><label>Top Brand</label>
</td></tr>
<tr><td>Visual:</td><td>
<input type=radio [% IF item.content==0 %]checked[% END %] name=visual value=0><label>Shaky or flashing</label>
<input type=radio [% IF item.content==1 %]checked[% END %] name=visual value=1><label>Body (bully button etc.)</label>
<input type=radio [% IF item.content==2 %]checked[% END %] name=visual value=2><label>Blank or partial blank</label>
<input type=radio [% IF item.content==3 %]checked[% END %] name=visual value=3><label>Dialog</label>
<input type=radio [% IF item.content==4 %]checked[% END %] name=visual value=4><label>General ugly</label>
<input type=radio [% IF item.content==5 %]checked[% END %] name=visual value=5><label>Normal</label>
<input type=radio [% IF item.content==6 %]checked[% END %] name=visual value=6><label>Fit nicely</label>
</td></tr>
<tr><td>Action:</td><td>
<input type=radio [% IF item.act==0 %]checked[% END %] name=act value=0><label>Pupup</label>
<input type=radio [% IF item.act==1 %]checked[% END %] name=act value=1><label>Audio</label>
<input type=radio [% IF item.act==2 %]checked[% END %] name=act value=2><label>Expandable</label>
<input type=radio [% IF item.act==3 %]checked[% END %] name=act value=3><label>Game play</label>
<input type=radio [% IF item.act==4 %]checked[% END %] name=act value=4><label>Download</label>
<input type=radio [% IF item.act==5 %]checked[% END %] name=act value=5><label>Normal</label>
</td></tr>
<tr><td>User Action:</td><td>
<input type=radio [% IF item.useract==0 %]checked[% END %] name=useract value=0><label>Pupup</label>
<input type=radio [% IF item.useract==1 %]checked[% END %] name=useract value=1><label>Audio</label>
<input type=radio [% IF item.useract==2 %]checked[% END %] name=useract value=2><label>Expandable</label>
<input type=radio [% IF item.useract==3 %]checked[% END %] name=useract value=3><label>Game play</label>
<input type=radio [% IF item.useract==4 %]checked[% END %] name=useract value=4><label>Download</label>
<input type=radio [% IF item.useract==5 %]checked[% END %] name=useract value=5><label>Normal</label>
</td></tr>
<tr><td>Download:</td><td>
<input type=radio [% IF item.download==0 %]checked[% END %] name=download value=0><label>Executable</label>
<input type=radio [% IF item.download==1 %]checked[% END %] name=download value=1><label>Smileys/Cursors/Avatar</label>
<input type=radio [% IF item.download==2 %]checked[% END %] name=download value=2><label>Wallpaper</label>
<input type=radio [% IF item.download==3 %]checked[% END %] name=download value=3><label>Screen saver</label>
<input type=radio [% IF item.download==4 %]checked[% END %] name=download value=4><label>Unknown software</label>
<input type=radio [% IF item.download==5 %]checked[% END %] name=download value=5><label>Normal or no download</label>
</td></tr>
<tr><td>Loading Speed:</td><td>
<input type=radio [% IF item.speed==0 %]checked[% END %] name=speed value=0><label>Normal</label>
<input type=radio [% IF item.speed==1 %]checked[% END %] name=speed value=1><label>Slow</label>
</td></tr>
<tr><td>Post Click:</td><td>
<input type=radio [% IF item.postclick==0 %]checked[% END %] name=postclick value=0><label>Hangup</label>
<input type=radio [% IF item.postclick==1 %]checked[% END %] name=postclick value=1><label>Broken link</label>
<input type=radio [% IF item.postclick==2 %]checked[% END %] name=postclick value=2><label>Wrong site</label>
<input type=radio [% IF item.postclick==3 %]checked[% END %] name=postclick value=3><label>Poor site</label>
<input type=radio [% IF item.postclick==4 %]checked[% END %] name=postclick value=4><label>Normal</label>
<input type=radio [% IF item.postclick==5 %]checked[% END %] name=postclick value=5><label>Good site</label>
</td></tr>
