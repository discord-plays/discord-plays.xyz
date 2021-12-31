package main

import "gorm.io/gorm"

type ProjectItem struct {
	gorm.Model
	Code        *string
	Name        *string
	SubText     *string
	Description *string
	Invite      *string
	ImageAlt    *string
	Notion      *string
	Github      *string
}

func NewProjectItem(code, name, subText, description, invite, imageAlt, notion, github string) *ProjectItem {
	return &ProjectItem{
		Code:        &code,
		Name:        &name,
		SubText:     &subText,
		Description: &description,
		Invite:      &invite,
		ImageAlt:    &imageAlt,
		Notion:      &notion,
		Github:      &github,
	}
}
