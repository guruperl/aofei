<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd"> 
<html xmlns="http://www.w3.org/1999/xhtml"> 
  <head> 
    <title>Edit Publisher Password</title> 
    <script src="../../../js/jquery-1.4.2.min.js"></script>
    [% INCLUDE form_ui_start.e %]
  <body> 
    <script>
      var url = new String( window.location )
      var SEARCH_STRING = "pubid="
      var idx = url.indexOf( SEARCH_STRING )
      var pair = url.substring( idx, url.length )
      var pubid = pair.split( '=' )[ 1 ] 

      $( document ).ready( 
        function() {
          $( '#pubid' )[ 0 ].value = pubid
          $( '#btnSubmit' )[ 0 ].value = "Change Password for Publisher #" + pubid
        }
      )      
    </script>
    <h2 align="center" class="curTitle" >Edit Publisher Password</h2>
    <div align="center">      
      <form action="pub" method="POST"> 
      <div id="container" style="width:500px;">
    <div id="mainmenu">
					<ul id="tabs">
						<li>
							<a href="#EditPublisher">Edit Publisher Password</a>
						</li>
					</ul>
				<div>
				<div class="bar">&nbsp;</div>
				<div class="panel" id="EditPublisher">
        <input id="action" name="action" type="hidden" value="updatepass" /> 
        <input id="pubid" name="pubid" type="hidden" /> 
        <table> 
          <tbody> 
            <tr> 
              <td> 
                <label>New password</label> 
              </td>          
              <td> 
                <input id="passwd" name="passwd" type="password" /> 
              </td> 
            </tr> 
            <tr> 
              <td> 
                <label>Confirm new password</label> 
              </td>          
              <td> 
                <input id="confirm" name="confirm" type="password" /> 
              </td>            
            </tr>          
          </tbody> 
        </table> 
        <br/> 
        <input id="btnSubmit" name="btnSubmit" type="submit" /> 
      </div><!--<div class="panel" id="EditPublisher">-->
      </div><!--<div id="container">-->
      </form>  
    </div> 
    [% INCLUDE form_ui_end.e %]
  </body> 
</html>