function applyOp (incoming_op) {
	committed_op.push(incoming_op);
    committed_string = committed_string.opAt(incoming_op.Type, incoming_op.Position, incoming_op.Value);
    var cursor_pos = getCursorPos($('#text')[0]).end;
    //modify local op
    if (incoming_op.ID == id) {
      local_op.shift();
      for (var i = 0; i < local_op.length; i++) {
        local_op[i].Version++;
      };
      if (sent == true) {sent = false};
    }else{
      //update local change
      for (var i = 0; i < local_op.length; i++) {
        local_op[i].Version++;

        if (incoming_op.Type == "Insert" && local_op[i].Type == "Insert") {
          if (incoming_op.Position <= local_op[i].Position) {
            local_op[i].Position = local_op[i].Position + 1;
          };
        }else if (incoming_op.Type == "Delete" && local_op[i].Type == "Insert") {
          if (incoming_op.Position < local_op[i].Position) {
            local_op[i].Position = local_op[i].Position - 1;
          };
        }else if (incoming_op.Type == "Insert" && local_op[i].Type == "Delete") {
          if (incoming_op.Position <= local_op[i].Position) {
            local_op[i].Position = local_op[i].Position + 1;
          };
        }else if (incoming_op.Type == "Delete" && local_op[i].Type == "Delete") {
          if (incoming_op.Position < local_op[i].Position) {
            local_op[i].Position = local_op[i].Position - 1;
          }else if (incoming_op.Position = local_op[i].Position) {
            if (sent == true) { //will be ignored on server, will not return
              local_op.splice(0,1);
              sent = false;
            }else{
              local_op.splice(0,1);
            };
          };

        };

      };

      //update cursor position
      if (incoming_op.Type == "Insert") {
        if (incoming_op.Position <= cursor_pos) {
          cursor_pos++;
        };
      }else if (incoming_op.Type == "Delete") {
        if (incoming_op.Position < cursor_pos) {
          cursor_pos--;
        };
      };


    };

    //update textarea view
    var show_text = committed_string;
    for (var i = 0; i < local_op.length; i++) {
      show_text = show_text.opAt(local_op[i].Type, local_op[i].Position, local_op[i].Value);
    };

    $("#text").val(show_text);
    setCaretToPos(document.getElementById("text"),cursor_pos);

    version_num++;
}

String.prototype.opAt = function(ind, index, c) {
  if (ind == "Insert") {
    return this.substr(0, index) + c + this.substr(index);
  }else if (ind == "Delete") {
    return this.substr(0, index) + this.substr(index+1);
  };
};

function sleep(ms) {
    var unixtime_ms = new Date().getTime();
    while(new Date().getTime() < unixtime_ms + ms) {}
}

function getCursorPos(input) {
    if ("selectionStart" in input && document.activeElement == input) {
        return {
            start: input.selectionStart,
            end: input.selectionEnd
        };
    }
    else if (input.createTextRange) {
        var sel = document.selection.createRange();
        if (sel.parentElement() === input) {
            var rng = input.createTextRange();
            rng.moveToBookmark(sel.getBookmark());
            for (var len = 0;
                     rng.compareEndPoints("EndToStart", rng) > 0;
                     rng.moveEnd("character", -1)) {
                len++;
            }
            rng.setEndPoint("StartToStart", input.createTextRange());
            for (var pos = { start: 0, end: len };
                     rng.compareEndPoints("EndToStart", rng) > 0;
                     rng.moveEnd("character", -1)) {
                pos.start++;
                pos.end++;
            }
            return pos;
        }
    }
    return -1;
}

function setSelectionRange(input, selectionStart, selectionEnd) {
  if (input.setSelectionRange) {
    input.focus();
    input.setSelectionRange(selectionStart, selectionEnd);
  }
  else if (input.createTextRange) {
    var range = input.createTextRange();
    range.collapse(true);
    range.moveEnd('character', selectionEnd);
    range.moveStart('character', selectionStart);
    range.select();
  }
}

function setCaretToPos (input, pos) {
  setSelectionRange(input, pos, pos);
}

