<tr><td colspan=2> <h3>Slot Coordinate</h3> </td></tr>
<tr><td>Membership:</td><td>
<input [% IF item.co_member=='Unknown'  %]checked[% END %] type=radio name=co_member value='Unknown'><label>Unknown</label>
<input [% IF item.co_member=='Yes'  %]checked[% END %] type=radio name=co_member value='Yes'><label>Yes</label>
<input [% IF item.co_member=='No'  %]checked[% END %] type=radio name=co_member value='No'><label>No</label>
</td></tr>
<tr><td>Window Frame:</td><td>
<input [% IF item.co_frame=="Unknown" %]checked[% END %] type=radio name=co_frame value='Unknown'><label>Unknown</label>
<input [% IF item.co_frame=="Normal" %]checked[% END %] type=radio name=co_frame value='Normal'><label>Normal</label>
<input [% IF item.co_frame=="SeparatedWindow" %]checked[% END %] type=radio name=co_frame value='SeparatedWindow'><label>Separated Window</label>
<input [% IF item.co_frame=="SeparatedJsWindow" %]checked[% END %] type=radio name=co_frame value='SeparatedJsWindow'><label>Separated Javascript Window</label>
</td></tr>
<tr><td>Page Level:</td><td>
<input [% IF item.co_pagelevel=="Unknown" %]checked[% END %] type=radio name=co_pagelevel value='Unknown'><label>Unknown</label>
<input [% IF item.co_pagelevel=="Homepage" %]checked[% END %] type=radio name=co_pagelevel value='Homepage'><label>Homepage</label>
<input [% IF item.co_pagelevel=="Section" %]checked[% END %] type=radio name=co_pagelevel value='Section'><label>Section</label>
<input [% IF item.co_pagelevel=="SubSection" %]checked[% END %] type=radio name=co_pagelevel value='SubSection'><label>Sub-Section</label>
<input [% IF item.co_pagelevel=="Rest" %]checked[% END %] type=radio name=co_pagelevel value='Rest'><label>Rest</label>
</td></tr>
<tr><td>Clock:</td><td>
<input [% IF item.co_clock=="Unknown" %]checked[% END %] type=radio name=co_clock value='Unknown'><label>Unknown</label>
<input [% IF item.co_clock==1 %]checked[% END %] type=radio name=co_clock value='1'><label>1</label>
<input [% IF item.co_clock==2 %]checked[% END %] type=radio name=co_clock value='2'><label>2</label>
<input [% IF item.co_clock==3 %]checked[% END %] type=radio name=co_clock value='3'><label>3</label>
<input [% IF item.co_clock==4 %]checked[% END %] type=radio name=co_clock value='4'><label>4</label>
<input [% IF item.co_clock==5 %]checked[% END %] type=radio name=co_clock value='5'><label>5</label>
<input [% IF item.co_clock==6 %]checked[% END %] type=radio name=co_clock value='6'><label>6</label>
<input [% IF item.co_clock==7 %]checked[% END %] type=radio name=co_clock value='7'><label>7</label>
<input [% IF item.co_clock==8 %]checked[% END %] type=radio name=co_clock value='8'><label>8</label>
<input [% IF item.co_clock==9 %]checked[% END %] type=radio name=co_clock value='9'><label>9</label>
<input [% IF item.co_clock==10 %]checked[% END %] type=radio name=co_clock value='10'><label>10</label>
<input [% IF item.co_clock==11 %]checked[% END %] type=radio name=co_clock value='11'><label>11</label>
<input [% IF item.co_clock==12 %]checked[% END %] type=radio name=co_clock value='12'><label>12</label>
</td></tr>
<tr><td>Y-Axis:</td><td>
<input [% IF item.co_yaxis=="Unknown" %]checked[% END %] type=radio name=co_yaxis value='Unknown'><label>Unknown</label>
<input [% IF item.co_yaxis=="ScrollUp" %]checked[% END %] type=radio name=co_yaxis value='ScrollUp'><label>Scroll Up</label>
<input [% IF item.co_yaxis=="ScrollDown" %]checked[% END %] type=radio name=co_yaxis value='ScrollDown'><label>Scroll Down</label>
<input [% IF item.co_yaxis=="ScrollMiddle" %]checked[% END %] type=radio name=co_yaxis value='ScrollMiddle'><label>Scroll Middle</label>
<input [% IF item.co_yaxis=="Sticky" %]checked[% END %] type=radio name=co_yaxis value='Sticky'><label>Sticky</label>
<input [% IF item.co_yaxis=="PopUnder" %]checked[% END %] type=radio name=co_yaxis value='PopUnder'><label>Pop Under</label>
<input [% IF item.co_yaxis=="JumpScreen" %]checked[% END %] type=radio name=co_yaxis value='JumpScreen'><label>Jump Screen</label>
</td></tr>
