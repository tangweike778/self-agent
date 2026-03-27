package skill

// AllSkills 获取所有技能
func AllSkills() []Skill {
	return []Skill{
		&ExecShellSkill{},
		&ReadFileSkill{},
		&WriteFileSkill{},
	}
}
