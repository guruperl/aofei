<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Edit Creative</title>
    <script src="../../../js/jquery-1.4.2.min.js"></script>
    [% INCLUDE form_ui_start.e %]
    <style type="text/css">
    	th{
    		text-align:right;
    	}
    	#container
    	{
    		width:600px;
    	}
    </style>

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
                  var count = 0              
                    
                  for ( ; i < OUTCASTS.length; i++ ) {
                    if ( this.value.indexOf( OUTCASTS[ i ] ) > -1 ) {
                      invalids.push( this.id )
                      break                      
                    }                      
                  }                  
                }
              )
              
              // check for invalid characters
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
    <div align="center">
     <h2 align="center" class="curTitle" >Edit Creative <a href="javascript:window.history.back()"><img src="/uilib/comImg/back.png" border=0 width="25" height="25" /></a></h2>
     <div id="container">
    <div id="mainmenu">
					<ul id="tabs">
						<li>
							<a href="#EditCreative">Edit Creative</a>
						</li>
					</ul>
				<div>
				<div class="bar">&nbsp;</div>
				<div class="panel" id="EditCreative">
      <form action="creative" method="POST">
        [% SET row = edit.0 %]
        <input id="action" name="action" type="hidden" value="update" />
        <input id="creativeid" name="creativeid" type="hidden" value="[% GET row.creativeid %]" />
        <input id="itemid" name="itemid" type="hidden" value="[% GET row.itemid %]" />        
        <table>
          <tbody id="[% GET row.creativeid %]">
            <tr>
              <th>Creative ID</th>
              <td style="text-align: right">[% GET row.creativeid %]</td>
            </tr>
            <tr>
              <th>Creative Name</th>
              <td>
                <input id="creativename" name="creativename" class="Editable" type="text" value="[% GET row.creativename %]" />
              </td>
            </tr>
            <tr>
              <th>Content</th>
              <td>
                <input id="content" name="content" class="Editable" type="text" value="[% GET row.content %]" />
              </td>
            </tr>
            <tr>
              <th>Status</th>
              <td>
                <select id="status" name="status" multiple="true" style="width:150px;">
                  [% IF row.status == 'Yes' %]
                    <option value="Yes" selected="true">Yes</option>
                    <option value="No">No</option>
                    <option value="Pause">Pause</option>
                  [% ELSIF row.status == 'No' %]
                    <option value="Yes">Yes</option>
                    <option value="No" selected="true">No</option>
                    <option value="Pause">Pause</option>
                  [% ELSIF row.status == 'Pause' %]  
                    <option value="Yes">Yes</option>
                    <option value="No">No</option>
                    <option value="Pause" selected="true">Pause</option>
                  [% END %]  
                </select>
              </td>
            </tr>  
            <tr>
            
            <td colspan="2" align="right" style="padding-right:20px;margin-right:20px;">
            <input type="reset" />
            &nbsp;  
            <input id="btnSubmit" type="submit" value="Save" />
            </td>
            </tr>
          </tbody>
        </table>              
        <br/>
        <div align="center">
          <textarea id="message" style="display:none;" cols="50" rows="2"></textarea>
        </div>        
      </form>
      </div><!--<div class="panel" id="EditCreative">-->
      </div> <!--<div id="container">-->
    </div> 
[% INCLUDE form_ui_end.e %] 
  </body>
</html>