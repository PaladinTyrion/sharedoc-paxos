function text_change () {
	var newVal = $("#text").val();

	//if (this.oldVal == null) {this.oldVal = "";};

	if (oldVal !== newVal) {
		//find the difference
		var type;
		var position;
		var value;

		if (oldVal.length < newVal.length) {
			type = "Insert";
			var found = false;
			for (var i = 0; i < oldVal.length; i++) {
				if (oldVal[i] !== newVal[i]) {
					position = i;
					value = newVal[position];
					found = true;
					break;
				};
			};
			if (found == false) {
				position = oldVal.length;
				value = newVal[position];
			};

		}else if (oldVal.length > newVal.length) {
			type = "Delete";
			var found = false;
			for (var i = 0; i < newVal.length; i++) {
				if (oldVal[i] !== newVal[i]) {
					position = i;
					value = oldVal[position];
					found = true;
					break;
				};
			};
			if (found == false) {
				position = oldVal.length-1;
				value = oldVal[position];
			};
		};

		//this.oldVal = newVal;
		if (type !== null && position !== null && value !== null) {
			var op = {ID: id, Version: version_num, Type: type, Position: position, Value: value};
			local_op.push(op);
		};
		

		//see results in console
		console.log(op);
		console.log(local_op);
		console.log(oldVal);
		console.log(newVal);

	};

}

function  CapsLock(e) {
  var Caps = null;
  var s = String.fromCharCode(e.which);
  if ( s.toUpperCase() === s && s.toLowerCase() !== s && !e.shiftKey ) {
    Caps = true;
  } else {
      Caps = false;
  }

  if(s.toUpperCase() === s){
      Caps = true;
  }else{
      Caps = false;
  }

  return Caps;
}