<tr><td colspan=2> <h3>Minimal Campaign Quality Accepted</h3> </td></tr>
<tr><td>Content:</td><td>
<select size=1 name=content>
<option [% IF item.content==0 %]selected[% END %] value=0>Top Brand</option>
<option [% IF item.content==1 %]selected[% END %] value=1>Good Brand</option>
<option [% IF item.content==2 %]selected[% END %] value=2>Normal</option>
<option [% IF item.content==3 %]selected[% END %] value=3>Unknown Dating</option>
<option [% IF item.content==6 %]selected[% END %] value=6>Provocative, Puzzle, Gambling</option>
<option [% IF item.content==7 %]selected[% END %] value=7>Bait</option>
</select>
</td></tr>
<tr><td>Visual:</td><td>
<select size=1 name=visual>
<option [% IF item.visual==0 %]selected[% END %] value=0>Fit Nicely</option>
<option [% IF item.visual==1 %]selected[% END %] value=1>Normal</option>
<option [% IF item.visual==3 %]selected[% END %] value=3>Ugly or Dialog</option>
<option [% IF item.visual==6 %]selected[% END %] value=6>Blank, Body Part, Shaky</option>
</select>
</td></tr>
<tr><td>Action:</td><td>
<select size=1 name=act>
<option [% IF item.act==0 %]selected[% END %] value=0>Normal</option>
<option [% IF item.act==1 %]selected[% END %] value=1>Download</option>
<option [% IF item.act==5 %]selected[% END %] value=5>Popup, Audio etc.</option>
</select>
</td></tr>
<tr><td>User Action:</td><td>
<select size=1 name=useract>
<option [% IF item.useract==0 %]selected[% END %] value=0>Normal</option>
<option [% IF item.useract==1 %]selected[% END %] value=1>Download</option>
<option [% IF item.useract==5 %]selected[% END %] value=5>Popup, Audio etc.</option>
</select>
</td></tr>
<tr><td>Download:</td><td>
<select size=1 name=download>
<option [% IF item.download==0 %]selected[% END %] value=0>Normal</option>
<option [% IF item.download==4 %]selected[% END %] value=4>Improper Content</option>
<option [% IF item.download==5 %]selected[% END %] value=5>Executable</option>
</select>
</td></tr>
<tr><td>Loading Speed:</td><td>
<select size=1 name=speed>
<option [% IF item.speed==0 %]selected[% END %] value=0>Normal</option>
<option [% IF item.speed==1 %]selected[% END %] value=1>Slow</option>
</select>
</td></tr>
<tr><td>Post Click:</td><td>
<select size=1 name=postclick>
<option [% IF item.postclick==0 %]selected[% END %] value=0>Good Site</option>
<option [% IF item.postclick==1 %]selected[% END %] value=1>Normal</option>
<option [% IF item.postclick==3 %]selected[% END %] value=3>Poor or Wrong</option>
<option [% IF item.postclick==5 %]selected[% END %] value=5>Broken</option>
</select>
</td></tr>
