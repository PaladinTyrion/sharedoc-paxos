function text_change () {
	var newVal = $("#text").val();

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

		}else if (this.oldVal.length > newVal.length) {
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

		oldVal = newVal;
		
		var op = {ID: id, Version: version_num, Type: type, Position: position, Value: value};
		local_op.push(op);

		//see results in console
		console.log(op);
		console.log(local_op);

	};

}