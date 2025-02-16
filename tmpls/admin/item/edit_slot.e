<tr><td colspan=2> <h3>Slots to Target</h3> </td></tr>
<tr><td>Membership:</td><td>
<input [% IF item.fl_m_any %]checked[% END %] type=checkbox name=fl_member value='Any'><label>Any</label>
<input [% IF item.fl_m_yes %]checked[% END %] type=checkbox name=fl_member value='Yes'><label>Yes</label>
<input [% IF item.fl_m_no %]checked[% END %] type=checkbox name=fl_member value='No'><label>No</label>
</td></tr>
<tr><td>Window Frame:</td><td>
<input [% IF item.fl_f_any %]checked[% END %] type=checkbox name=fl_frame value='Any'><label>Any</label>
<input [% IF item.fl_f_normal %]checked[% END %] type=checkbox name=fl_frame value='Normal'><label>Normal</label>
<input [% IF item.fl_f_separatedwindow %]checked[% END %] type=checkbox name=fl_frame value='SeparatedWindow'><label>Separated Window</label>
<input [% IF item.fl_f_separatedjswindow %]checked[% END %] type=checkbox name=fl_frame value='SeparatedJsWindow'><label>Separated Javascript Window</label>
</td></tr>
<tr><td>Page Level:</td><td>
<input [% IF item.fl_p_any %]checked[% END %] type=checkbox name=fl_pagelevel value='Any'><label>Any</label>
<input [% IF item.fl_p_homepage %]checked[% END %] type=checkbox name=fl_pagelevel value='Homepage'><label>Homepage</label>
<input [% IF item.fl_p_section %]checked[% END %] type=checkbox name=fl_pagelevel value='Section'><label>Section</label>
<input [% IF item.fl_p_subsection %]checked[% END %] type=checkbox name=fl_pagelevel value='SubSection'><label>Sub-Section</label>
<input [% IF item.fl_p_rest %]checked[% END %] type=checkbox name=fl_pagelevel value='Rest'><label>Rest</label>
</td></tr>
<tr><td>Clock:</td><td>
<input [% IF item.fl_c_any %]checked[% END %] type=checkbox name=fl_clock value='Any'><label>Any</label>
<input [% IF item.fl_c_1 %]checked[% END %] type=checkbox name=fl_clock value='1'><label>1</label>
<input [% IF item.fl_c_2 %]checked[% END %] type=checkbox name=fl_clock value='2'><label>2</label>
<input [% IF item.fl_c_3 %]checked[% END %] type=checkbox name=fl_clock value='3'><label>3</label>
<input [% IF item.fl_c_4 %]checked[% END %] type=checkbox name=fl_clock value='4'><label>4</label>
<input [% IF item.fl_c_5 %]checked[% END %] type=checkbox name=fl_clock value='5'><label>5</label>
<input [% IF item.fl_c_6 %]checked[% END %] type=checkbox name=fl_clock value='6'><label>6</label>
<input [% IF item.fl_c_7 %]checked[% END %] type=checkbox name=fl_clock value='7'><label>7</label>
<input [% IF item.fl_c_8 %]checked[% END %] type=checkbox name=fl_clock value='8'><label>8</label>
<input [% IF item.fl_c_9 %]checked[% END %] type=checkbox name=fl_clock value='9'><label>9</label>
<input [% IF item.fl_c_10 %]checked[% END %] type=checkbox name=fl_clock value='10'><label>10</label>
<input [% IF item.fl_c_11 %]checked[% END %] type=checkbox name=fl_clock value='11'><label>11</label>
<input [% IF item.fl_c_12 %]checked[% END %] type=checkbox name=fl_clock value='12'><label>12</label>
</td></tr>
<tr><td>Y-Axis:</td><td>
<input [% IF item.fl_y_any %]checked[% END %] type=checkbox name=fl_yaxis value='Any'><label>Any</label>
<input [% IF item.fl_y_scrollup %]checked[% END %] type=checkbox name=fl_yaxis value='ScrollUp'><label>Scroll Up</label>
<input [% IF item.fl_y_scrolldown %]checked[% END %] type=checkbox name=fl_yaxis value='ScrollDown'><label>Scroll Down</label>
<input [% IF item.fl_y_scrollmiddle %]checked[% END %] type=checkbox name=fl_yaxis value='ScrollMiddle'><label>Scroll Middle</label>
<input [% IF item.fl_y_sticky %]checked[% END %] type=checkbox name=fl_yaxis value='Sticky'><label>Sticky</label>
<input [% IF item.fl_y_popunder %]checked[% END %] type=checkbox name=fl_yaxis value='PopUnder'><label>Pop Under</label>
<input [% IF item.fl_y_jumpscreen %]checked[% END %] type=checkbox name=fl_yaxis value='JumpScreen'><label>Jump Screen</label>
</td></tr>
