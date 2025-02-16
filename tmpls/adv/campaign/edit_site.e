<tr><td colspan=2> <h3>Sites to Target</h3> </td></tr>
<tr><td>Language:</td><td><select size=1 name=languageid>
<option [% IF item.languageid==0 %]selected[% END %] value=0>Adjusted</option>
<option [% IF item.languageid==1 %]selected[% END %] value=1>English</option>
<option [% IF item.languageid==2 %]selected[% END %] value=2>Arabic</option>
<option [% IF item.languageid==3 %]selected[% END %] value=3>Chinese</option>
<option [% IF item.languageid==4 %]selected[% END %] value=4>French</option>
<option [% IF item.languageid==5 %]selected[% END %] value=5>Russian</option>
<option [% IF item.languageid==6 %]selected[% END %] value=6>Spanish</option>
</select></td></tr>
<tr><td>Platform:</td><td>
<input type=checkbox [% IF item.fl_p_web %]checked[% END %] name=fl_platform value='Web'><label>Web</label>
<input type=checkbox [% IF item.fl_p_mobile %]checked[% END %] name=fl_platform value='Mobile'><label>Mobile</label>
<input type=checkbox [% IF item.fl_p_email %]checked[% END %] name=fl_platform  value='Email'><label>Email</label>
<input type=checkbox [% IF item.fl_p_video %]checked[% END %] name=fl_platform  value='Video'><label>Video</label>
<input type=checkbox [% IF item.fl_p_device %]checked[% END %] name=fl_platform value='Device'><label>Device</label>
</td></tr>
<tr><td>Reader Group:</td><td>
<input type=checkbox [% IF item.fl_r_men %]checked[% END %] name=fl_reader value='Men'><label>Men</label>
<input type=checkbox [% IF item.fl_r_women %]checked[% END %] name=fl_reader value='Women'><label>Women</label>
<input type=checkbox [% IF item.fl_r_teens %]checked[% END %] name=fl_reader value='Teens'><label>Teens</label>
<input type=checkbox [% IF item.fl_r_kids %]checked[% END %] name=fl_reader value='Kids'><label>Kids</label>
</td></tr>
<tr><td>Site Type:</td><td>
<input type=checkbox [% IF item.fl_s_content %]checked[% END %] name=fl_style value='Content'><label>Content</label>
<input type=checkbox [% IF item.fl_s_blog %]checked[% END %] name=fl_style value='Blog'><label>Blog</label>
<input type=checkbox [% IF item.fl_s_directory %]checked[% END %] name=fl_style value='Directory'><label>Directory</label>
<input type=checkbox [% IF item.fl_s_ecommerce %]checked[% END %] name=fl_style value='Ecommerce'><label>Ecommerce</label>
<input type=checkbox [% IF item.fl_s_social %]checked[% END %] name=fl_style value='Social'><label>Social Network</label>
<input type=checkbox [% IF item.fl_s_search %]checked[% END %] name=fl_style value='Search'><label>Search</label>
</td></tr>
<tr><td>Sale Type:</td><td>
<input type=checkbox [% IF item.fl_v_direct %]checked[% END %] name=fl_vertical value='Direct'><label>Direct</label>
<input type=checkbox [% IF item.fl_v_indirect %]checked[% END %] name=fl_vertical value='Indirect'><label>Indirect</label>
</td></tr>
