// Fixie teaching-mode explanations, shown by teach.js. Created by nrmluvdev.
window.TEACH = {
  app: "Fixie",
  intro: "Point at any button, tab or box in Fixie to learn what it does.",
  tips: {
    // top bar
    "Repair tab|Repair": ["Repair", "The step-by-step guide. Pick what you're fixing and what it's doing, then Fixie takes you through one check at a time until it reaches the likely fault."],
    "Tools tab|Tools": ["Tools", "Every tool the guide mentions: what it's for, roughly what it costs and how to use it. Tick the ones you have."],
    "Signs tab|What to look for": ["What to look for", "Pictures and descriptions of failed parts: bulging capacitors, burnt parts, corrosion, cracked joints, blown fuses and swollen batteries."],
    "Safety tab|Safety": ["Safety", "How to work safely with mains, big capacitors, lithium batteries and soldering, and the things you should never open at home."],
    "My repairs tab|My repairs": ["My repairs", "Every repair you've started, saved on this computer as you go. Open one to carry on where you left off, or save its report."],
    "Start a new repair|New repair": ["New repair", "Starts a new repair. The one you were working on stays saved in My repairs."],
    "Save the repair report|Save report|Report": ["Save report", "Saves the repair as a text file: every step, your answers, readings and notes, and the fault and fix it found. Handy for a repair shop, or a parts order."],
    "Teaching tips (F2)": ["Teaching tips", "Turns these explanations on or off. The F2 key does the same."],
    // picking
    "Device name": ["What is it?", "Type what you're fixing, in your own words. It names the repair, and Fixie checks it against the things it won't help with for safety (microwaves, ovens, toasters, heaters and so on)."],
    "Closest kind": ["Closest kind of thing", "A shortcut: picking one fills in the usual power and features below, which you can then change. If nothing fits, pick Something else."],
    "How it's powered": ["How it's powered", "Pick every way it gets power. This decides which power checks Fixie gives you: socket and fuse for mains, output voltage for adapters, current for USB, cells for batteries, battery health for built-in batteries."],
    "What it has": ["What it has", "Pick everything it has. Fixie only offers the symptoms and checks that fit: no picture checks without a screen, no motor checks without a motor."],
    "Next: symptoms|Next: what's it doing?": ["Next", "Goes on to choose what's wrong with it."],
    "Not covered|What Fixie won't help with": ["What Fixie won't help with", "For safety, Fixie gives no repair steps for microwaves, anything that heats from the mains, large appliances, high-voltage devices, house wiring and gas, big batteries, and medical or life-safety devices."],
    "Pick a symptom": ["What's it doing?", "Pick the problem that fits best. Only the symptoms that apply to this kind of device are shown."],
    "Resume repair|Carry on": ["Carry on", "Opens the repair you were last working on, at the step you'd reached."],
    // the step
    "Repair name": ["Repair name", "Give the repair a name you'll recognise later, like 'Kitchen kettle' or 'Sam's controller'."],
    "Your path": ["Your path", "Every step you've been through, with your answers and readings. Click an earlier step to go back to it and choose differently."],
    "Tools for this step": ["Tools", "The tools this step uses. Click one to see how to use it. A tick means you have it; a dashed outline means it's not in the basic starter kit."],
    "Signs for this step": ["What it looks like", "Click to see a picture of a good part next to a faulty one."],
    "Your reading": ["Your reading", "Type in what your meter shows. Fixie says whether it's good or bad, and highlights the answer that matches."],
    "Answers": ["Answers", "Pick what you found. The number keys 1 to 9 choose an answer too."],
    "Safety check": ["Safety check", "Steps inside a device that runs from the mains, or has a lithium battery, need it made safe first. Tick this once you have."],
    "Step notes": ["Notes", "Anything worth remembering: readings, part numbers, where screws went. Notes go in the saved report."],
    "Go back a step (Alt+Left)|Back|Try a different answer": ["Back", "Goes back to the step before, so you can choose a different answer. Alt+Left does the same."],
    "Start this repair again|Start again": ["Start again", "Goes back to the first step of this repair."],
    // results
    "Fixed it": ["Fixed it", "Marks the repair as fixed in My repairs."],
    "Not fixed": ["Keep going", "If the fix didn't work, this takes you on to the next check."],
    "Stop here": ["Stop here", "Marks the repair as stopped (perhaps it's going to a repair shop). It stays in My repairs."],
    "How hard the fix is": ["How hard it is", "Easy: no special skills. Medium: some soldering or careful dismantling. Hard: fine soldering or board-level work. Pro: best left to a repair shop or electrician."],
    // reference
    "Starter kit": ["Starter kit", "How many of the basic tools you have. These cover most repairs."],
    "Open repair|Open": ["Open", "Opens this repair at the step you'd reached."],
    "Delete repair|Delete": ["Delete", "Removes this repair. Click twice to make sure."],
    "Look & accessibility": ["Look & accessibility", "Themes, accent colour, text size, touch-friendly controls, reduced motion and other settings for this app. Ctrl+, opens it too."],
  },
};
