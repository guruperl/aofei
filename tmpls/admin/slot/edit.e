<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Edit Slot</title>
    <script src="../../../js/jquery-1.4.2.min.js"></script>
     [% INCLUDE form_ui_start.e %]
  </head>
<body>
  <script>
    $( document ).ready(
      function() {        
        $( "#btnSubmit" ).click(
          function() {
            var OUTCASTS = [ '!', '#', '$', '%', '^', '|', '{', '}' ]
            var ERROR_GENERAL = "Invalid parameter value(s):\n"              
            var invalids = new Array()
            var mve = new String()

            // check for invalid input
            $( ".Editable" ).each(
              function() {                  
                var i = 0    
                    
                for ( ; i < OUTCASTS.length; i++ ) {
                  if ( this.value.indexOf( OUTCASTS[ i ] ) > -1 ) {
                    invalids.push( this.id )
                    break                      
                  }                      
                }                  
              }
            )
              
            if ( invalids.length > 0 ) {
              mve = ERROR_GENERAL + invalids              
              $( "#message" )[ 0 ].value = mve
              return false
            }
          }
        ) 
      }  
    )  
  </script>
    [% SET item = edit.0 %]
    <h2 align="center" class="curTitle" >Edit Slot <a href="javascript:window.history.back()"><img src="/uilib/comImg/back.png" border=0 width="25" height="25" /></a></h2>
    <div>
    <div id="container">
    <div id="mainmenu">
					<ul id="tabs">
						<li>
							<a href="#EditSlot">Edit Slot</a>
						</li>
					</ul>
				<div>
				<div class="bar">&nbsp;</div>
				<div class="panel" id="EditSlot">
      <form action="slot" method="POST">
        <input id="action" name="action" type="hidden" value="update" />
        <input id="slotid" name="slotid" type="hidden" value="[% GET item.slotid %]" />
        <input id="siteid" name="siteid" type="hidden" value=[% GET item.siteid %] />
        <h2>Edit Slot</h2>
        <table>          
          <tbody>
            <tr>
              <td>Slot ID:</td>
              <td>[% GET item.slotid %]</td>
            </tr>          
            <tr>
              <td>Slot Name:</td>
              <td>
                <input id="slotname" name="slotname" class="Editable" type="text" value="[% GET item.slotname %]" />
              </td>
            </tr>
           <tr>
             <table>
               <tr>
                 <td>Channel&nbsp;Order:</td>
                 <td>
                   <select id="channelorder" name="channelorder" multiple="true">
                     <option [% IF item.channelorder == 'AllowDeny' %] selected="true" [% END %] value="AllowDeny">AllowDeny</option>
                     <option [% IF item.channelorder == 'DenyAllow' %] selected="true" [% END %] value="DenyAllow">DenyAllow</option>      
                     <option [% IF item.channelorder == 'No' %] selected="true" [% END %] value="No">No</option>
                   </select>
                 </td>
                 <td>Access&nbsp;Order:</td>
                 <td>
                   <select id="accessorder" name="accessorder" multiple="true">
                     <option [% IF item.accessorder == 'AllowDeny' %] selected="true" [% END %] value="AllowDeny">AllowDeny</option>
                     <option [% IF item.accessorder == 'DenyAllow' %] selected="true" [% END %] value="DenyAllow">DenyAllow</option>      
                     <option [% IF item.accessorder == 'No' %] selected="true" [% END %] value="No">No</option>
                   </select>
                 </td>
                 <td>Status:</td>
                 <td>
                   <select id="status" name="status" multiple="true">
                     <option [% IF item.status == 'Yes' %] selected="true" [% END %] value="Yes">Yes</option>
                     <option [% IF item.status == 'No' %] selected="true" [% END %] value="No">No</option>
                     <option [% IF item.status == 'New' %] selected="true" [% END %] value="New">New</option>
                   </select>
                 </td>
               </tr>
             </table>
           </tr>            
            <tr>
              <td>Membership:</td>
              <td>
               <input [% IF item.fl_m_any %]checked[% END %] type=checkbox name=fl_member value='Any' />
               <label>Any</label>
               <input [% IF item.fl_m_yes %]checked[% END %] type=checkbox name=fl_member value='Yes'>
               <label>Yes</label>
               <input [% IF item.fl_m_no %]checked[% END %] type=checkbox name=fl_member value='No'>
               <label>No</label>
             </td>
           </tr>
           <tr>
             <td>Window Frame:</td>
             <td>
               <input [% IF item.fl_f_any %]checked[% END %] type=checkbox name=fl_frame value='Any' />
               <label>Any</label>
               <input [% IF item.fl_f_normal %]checked[% END %] type=checkbox name=fl_frame value='Normal' />
               <label>Normal</label>
               <input [% IF item.fl_f_separatedwindow %]checked[% END %] type=checkbox name=fl_frame value='SeparatedWindow' />
               <label>Separated Window</label>
               <input [% IF item.fl_f_separatedjswindow %]checked[% END %] type=checkbox name=fl_frame value='SeparatedJsWindow' />
               <label>Separated Javascript Window</label>
             </td>
           </tr>
           <tr>
             <td>Page Level:</td>
             <td>           
               <input [% IF item.fl_p_any %]checked[% END %] type=checkbox name=fl_pagelevel value='Any' />
               <label>Any</label>
               <input [% IF item.fl_p_homepage %]checked[% END %] type=checkbox name=fl_pagelevel value='Homepage' />
               <label>Homepage</label>
               <input [% IF item.fl_p_section %]checked[% END %] type=checkbox name=fl_pagelevel value='Section' />
               <label>Section</label>
               <input [% IF item.fl_p_subsection %]checked[% END %] type=checkbox name=fl_pagelevel value='SubSection' />
               <label>Sub-Section</label>
               <input [% IF item.fl_p_rest %]checked[% END %] type=checkbox name=fl_pagelevel value='Rest' />
               <label>Rest</label>
             </td>
           </tr>
           <tr>
             <td>Clock:</td>
             <td>
               <input [% IF item.fl_c_any %]checked[% END %] type=checkbox name=fl_clock value='Any' />
               <label>Any</label>
               <input [% IF item.fl_c_1 %]checked[% END %] type=checkbox name=fl_clock value='1' />
               <label>1</label>
               <input [% IF item.fl_c_2 %]checked[% END %] type=checkbox name=fl_clock value='2' />
               <label>2</label>
               <input [% IF item.fl_c_3 %]checked[% END %] type=checkbox name=fl_clock value='3' />
               <label>3</label>
               <input [% IF item.fl_c_4 %]checked[% END %] type=checkbox name=fl_clock value='4' />
               <label>4</label>
               <input [% IF item.fl_c_5 %]checked[% END %] type=checkbox name=fl_clock value='5' />
               <label>5</label>
               <input [% IF item.fl_c_6 %]checked[% END %] type=checkbox name=fl_clock value='6' />
               <label>6</label>
               <input [% IF item.fl_c_7 %]checked[% END %] type=checkbox name=fl_clock value='7' />
               <label>7</label>
               <input [% IF item.fl_c_8 %]checked[% END %] type=checkbox name=fl_clock value='8' />
               <label>8</label>
               <input [% IF item.fl_c_9 %]checked[% END %] type=checkbox name=fl_clock value='9' />
               <label>9</label>
               <input [% IF item.fl_c_10 %]checked[% END %] type=checkbox name=fl_clock value='10' />
               <label>10</label>
               <input [% IF item.fl_c_11 %]checked[% END %] type=checkbox name=fl_clock value='11' />
               <label>11</label>
               <input [% IF item.fl_c_12 %]checked[% END %] type=checkbox name=fl_clock value='12' />
               <label>12</label>
             </td>
           </tr>
           <tr>
             <td>Y-Axis:</td>
             <td>
               <input [% IF item.fl_y_any %]checked[% END %] type=checkbox name=fl_yaxis value='Any' />
               <label>Any</label>
               <input [% IF item.fl_y_scrollup %]checked[% END %] type=checkbox name=fl_yaxis value='ScrollUp' />
               <label>Scroll Up</label>
               <input [% IF item.fl_y_scrolldown %]checked[% END %] type=checkbox name=fl_yaxis value='ScrollDown' />
               <label>Scroll Down</label>
               <input [% IF item.fl_y_scrollmiddle %]checked[% END %] type=checkbox name=fl_yaxis value='ScrollMiddle' />
               <label>Scroll Middle</label>
               <input [% IF item.fl_y_sticky %]checked[% END %] type=checkbox name=fl_yaxis value='Sticky' />
               <label>Sticky</label>
               <input [% IF item.fl_y_popunder %]checked[% END %] type=checkbox name=fl_yaxis value='PopUnder' />
               <label>Pop Under</label>
               <input [% IF item.fl_y_jumpscreen %]checked[% END %] type=checkbox name=fl_yaxis value='JumpScreen' />
               <label>Jump Screen</label>
             </td>
           </tr>
         </tbody>
       </table>
       <br/>    
       <div align="center">
         <input type="reset" value="Reset" />
         &nbsp;
         <input id="btnSubmit" type="submit" value="Update Slot" />
         <br/>
         <br/>
         <textarea id="message" cols="50" rows="2" style="display:none;"></textarea>
       </div>
     </form> 
     </div><!--<div class="panel" id="EditSlot">-->
      </div><!--<div id="container">--> 
   </div>
   [% INCLUDE form_ui_end.e %]
</body>

